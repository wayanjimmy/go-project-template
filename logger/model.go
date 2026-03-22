package logger

import (
	"context"
	"log/slog"
	"time"
)

type Level slog.Level

const (
	LevelInfo  = Level(slog.LevelInfo)
	LevelError = Level(slog.LevelError)
	LevelDebug = Level(slog.LevelDebug)
)

type Record struct {
	Time       time.Time
	Message    string
	Level      Level
	Attributes map[string]any
}

func toRecord(r slog.Record) Record {
	atts := make(map[string]any, r.NumAttrs())

	r.Attrs(func(attr slog.Attr) bool {
		atts[attr.Key] = attr.Value.Any()
		return true
	})

	return Record{
		Time:       r.Time,
		Message:    r.Message,
		Level:      Level(r.Level),
		Attributes: atts,
	}
}

type EventFn func(ctx context.Context, r Record)

type Events struct {
	Info  EventFn
	Error EventFn
}
