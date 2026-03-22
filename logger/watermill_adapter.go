package logger

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
)

// WatermillAdapter maps Watermill logging to this package's JSON logger.
// Debug and Trace are intentionally mapped to Info.
type WatermillAdapter struct {
	log    *Logger
	fields watermill.LogFields
}

func NewWatermillAdapter(log *Logger) watermill.LoggerAdapter {
	return &WatermillAdapter{
		log:    log,
		fields: watermill.LogFields{"component": "watermill"},
	}
}

func (a *WatermillAdapter) Error(msg string, err error, fields watermill.LogFields) {
	merged := a.fields.Add(fields)
	args := toArgs(merged)
	if err != nil {
		args = append(args, "error", err.Error())
	}

	a.log.Error(context.Background(), msg, args...)
}

func (a *WatermillAdapter) Info(msg string, fields watermill.LogFields) {
	merged := a.fields.Add(fields)
	a.log.Info(context.Background(), msg, toArgs(merged)...)
}

func (a *WatermillAdapter) Debug(msg string, fields watermill.LogFields) {
	merged := a.fields.Add(fields)
	a.log.Info(context.Background(), msg, toArgs(merged)...)
}

func (a *WatermillAdapter) Trace(msg string, fields watermill.LogFields) {
	merged := a.fields.Add(fields)
	a.log.Info(context.Background(), msg, toArgs(merged)...)
}

func (a *WatermillAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &WatermillAdapter{
		log:    a.log,
		fields: a.fields.Add(fields),
	}
}

func toArgs(fields watermill.LogFields) []any {
	if len(fields) == 0 {
		return nil
	}

	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	return args
}
