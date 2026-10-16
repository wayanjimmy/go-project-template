// Package ntfy sends alert notifications to an ntfy server (https://ntfy.sh).
//
// Sends happen on a bounded worker pool with per-second rate limiting and a
// circuit breaker so a flood of error logs cannot overwhelm the server or the
// process itself.
package ntfy

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"
)

// Client publishes messages to an ntfy server.
type Client struct {
	workers *pool.Pool
	pending chan struct{}
	closed  atomic.Bool

	server string
	topic  string
	token  string
	http   *http.Client

	mu                  sync.Mutex
	lastSent            time.Time
	consecutiveFailures int
	circuitOpenUntil    time.Time
}

// NewClient creates a client for the ntfy server at serverURL (e.g.
// http://localhost:25999), publishing to topic. token may be empty for
// servers without access control.
func NewClient(serverURL, topic, token string) *Client {
	return &Client{
		workers: pool.New().WithMaxGoroutines(10),
		pending: make(chan struct{}, 100),
		server:  strings.TrimSuffix(serverURL, "/"),
		topic:   topic,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Send publishes message asynchronously. Alerts are dropped when the queue is
// full, the circuit breaker is open, or the per-second limit is reached.
func (c *Client) Send(ctx context.Context, message string) {
	if c.closed.Load() {
		return
	}

	if !c.allowSend() {
		return
	}

	select {
	case c.pending <- struct{}{}:
	default:
		// Drop alerts when queue is full to avoid unbounded memory growth.
		return
	}

	c.workers.Go(func() {
		defer func() { <-c.pending }()

		if err := c.deliver(ctx, message); err != nil {
			c.onFailure(err)
			return
		}
		c.onSuccess()
	})
}

// Close waits for in-flight sends to finish, up to ctx.
func (c *Client) Close(ctx context.Context) error {
	c.closed.Store(true)

	// Wait for in-flight sends on a separate goroutine so Close can honor
	// ctx timeouts; pool.Wait blocks until every spawned worker returns.
	done := make(chan struct{})
	go func() {
		c.workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) allowSend() bool {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if now.Before(c.circuitOpenUntil) {
		return false
	}

	if now.Sub(c.lastSent) < time.Second {
		return false
	}

	c.lastSent = now
	return true
}

func (c *Client) deliver(ctx context.Context, message string) error {
	if message == "" {
		return fmt.Errorf("empty ntfy message")
	}
	if c.server == "" || c.topic == "" {
		return fmt.Errorf("ntfy server or topic not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server+"/"+c.topic, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("create ntfy request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send ntfy message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy server returned %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) onSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.consecutiveFailures = 0
	c.circuitOpenUntil = time.Time{}
}

func (c *Client) onFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Printf("ntfy client: send failed: %v\n", err)
	c.consecutiveFailures++

	if c.consecutiveFailures >= 3 {
		c.circuitOpenUntil = time.Now().Add(30 * time.Second)
	}
}
