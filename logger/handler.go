package logger

import (
	"context"
	"log/slog"
)

type logHandler struct {
	handler slog.Handler
	events  Events
}

func newLogHandler(handler slog.Handler, events Events) *logHandler {
	return &logHandler{
		handler: handler,
		events:  events,
	}
}

// Enabled implements slog.Handler.
func (l *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return l.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (l *logHandler) Handle(ctx context.Context, r slog.Record) error {
	switch r.Level {
	case slog.LevelInfo:
		if l.events.Info != nil {
			l.events.Info(ctx, toRecord(r))
		}
	case slog.LevelError:
		if l.events.Error != nil {
			l.events.Error(ctx, toRecord(r))
		}
	}

	return l.handler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (l *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &logHandler{
		handler: l.handler.WithAttrs(attrs),
		events:  l.events,
	}
}

// WithGroup implements slog.Handler.
func (l *logHandler) WithGroup(name string) slog.Handler {
	return &logHandler{
		handler: l.handler.WithGroup(name),
		events:  l.events,
	}
}
