package gchat

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"
)

type Client struct {
	workers *pool.Pool
	pending chan struct{}
	closed  atomic.Bool

	mu                  sync.Mutex
	lastSent            time.Time
	consecutiveFailures int
	circuitOpenUntil    time.Time
}

func NewClient() *Client {
	return &Client{
		workers: pool.New().WithMaxGoroutines(10),
		pending: make(chan struct{}, 100),
	}
}

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

func (c *Client) Close(ctx context.Context) error {
	c.closed.Store(true)

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

func (c *Client) deliver(_ context.Context, message string) error {
	if message == "" {
		return fmt.Errorf("empty gchat message")
	}

	log.Println("gchat client: sending message")
	defer log.Println("gchat client: message sent")
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

	log.Printf("gchat client: send failed: %v\n", err)
	c.consecutiveFailures++

	if c.consecutiveFailures >= 3 {
		c.circuitOpenUntil = time.Now().Add(30 * time.Second)
	}
}
