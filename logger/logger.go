package logger

import (
	"context"
	"fmt"
	"go-project-template/buildinfo"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"
)

type CorrelationIDFn func(ctx context.Context) string

type Logger struct {
	handler         slog.Handler
	correlationIDFn CorrelationIDFn
}

func New(w io.Writer, minLevel Level, serviceName string, correlationIDFn CorrelationIDFn) *Logger {
	return new(w, minLevel, serviceName, correlationIDFn, Events{})
}

func NewWithEvents(w io.Writer, minLevel Level, serviceName string, correlationIDFn CorrelationIDFn, events Events) *Logger {
	return new(w, minLevel, serviceName, correlationIDFn, events)
}

func NewWithHandler(h slog.Handler) *Logger {
	return &Logger{handler: h}
}

func Noop() *Logger {
	return &Logger{
		handler: slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.Level(LevelError) + 1}),
	}
}

func (log *Logger) Info(ctx context.Context, msg string, args ...any) {
	log.write(ctx, LevelInfo, 3, msg, args...)
}

func (log *Logger) Error(ctx context.Context, msg string, args ...any) {
	log.write(ctx, LevelError, 3, msg, args...)
}

func (log *Logger) write(ctx context.Context, level Level, caller int, msg string, args ...any) {
	slogLevel := slog.Level(level)

	if !log.handler.Enabled(ctx, slogLevel) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(caller, pcs[:])

	r := slog.NewRecord(time.Now(), slogLevel, msg, pcs[0])

	if log.correlationIDFn != nil {
		if id := log.correlationIDFn(ctx); id != "" {
			args = append(args, "correlation_id", id)
		}
	}
	r.Add(args...)

	log.handler.Handle(ctx, r)
}

func new(w io.Writer, minLevel Level, serviceName string, correlationIDFn CorrelationIDFn, events Events) *Logger {
	f := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			if source, ok := a.Value.Any().(*slog.Source); ok {
				v := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
				return slog.Attr{Key: "file", Value: slog.StringValue(v)}
			}
		}

		return a
	}

	handler := slog.Handler(slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: true, Level: slog.Level(minLevel), ReplaceAttr: f}))

	if events.Info != nil || events.Error != nil {
		handler = newLogHandler(handler, events)
	}

	attrs := []slog.Attr{
		{Key: "service", Value: slog.StringValue(serviceName)},
		{Key: "version", Value: slog.StringValue(buildinfo.Version)},
	}

	handler = handler.WithAttrs(attrs)

	return &Logger{
		handler:         handler,
		correlationIDFn: correlationIDFn,
	}
}
