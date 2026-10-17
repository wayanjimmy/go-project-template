// Package alerting wires the error-log alerting client into the application
// logger. When NTFY_URL (and optionally NTFY_TOPIC / NTFY_TOKEN) are set,
// every error-level log line is sent as an ntfy notification.
package alerting

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"go-project-template/buildinfo"
	"go-project-template/client/ntfy"
	"go-project-template/logger"
	"go-project-template/requestid"
)

// AlertSender is the subset of the notification client used by the logger.
type AlertSender interface {
	Send(ctx context.Context, message string)
	Close(ctx context.Context) error
}

// NewLogger returns an application logger that notifies ntfy on error-level
// logs. The returned close function flushes pending notifications; call it
// during shutdown. It is always non-nil and safe to call unconditionally.
func NewLogger(ctx context.Context, w io.Writer) (*logger.Logger, func()) {
	client := ntfy.NewClient(
		os.Getenv("NTFY_URL"),
		envOrDefault("NTFY_TOPIC", "app-alerts"),
		os.Getenv("NTFY_TOKEN"),
	)

	events := logger.Events{}
	var sender AlertSender = client

	if os.Getenv("NTFY_URL") == "" {
		// No ntfy configured; never send.
		sender = nil
	} else {
		events.Error = func(_ context.Context, r logger.Record) {
			// Detach from the request context: the HTTP request that logged
			// the error may finish (cancelling its context) before the async
			// delivery goroutine runs.
			sender.Send(context.WithoutCancel(ctx), formatRecord(buildinfo.Service, r))
		}
	}

	log := logger.NewWithEvents(w, logger.LevelInfo, buildinfo.Service, requestid.FromContext, events)

	return log, func() {
		if sender == nil {
			return
		}
		_ = sender.Close(ctx)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// formatRecord renders a log record as a single-line alert message, e.g.
// "APP-WORKER: startup: error=invalid config: ...".
func formatRecord(service string, r logger.Record) string {
	var b strings.Builder
	b.WriteString(service)
	b.WriteString(": ")
	b.WriteString(r.Message)
	for k, v := range r.Attributes {
		fmt.Fprintf(&b, "\n%s=%v", k, v)
	}
	return b.String()
}
