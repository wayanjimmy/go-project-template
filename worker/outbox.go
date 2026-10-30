package worker

import (
	"context"
	"go-project-template/logger"
	"go-project-template/repository"
	"go-project-template/internal/requestid"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
)

func RunOutbox(ctx context.Context, db *sqlx.DB, pub message.Publisher, log *logger.Logger) error {
	return Poll(ctx, log, "outbox", func(ctx context.Context) error {
		return repository.RelayOutbox(ctx, db, func(ev repository.OutboxEvent) error {
			msg := message.NewMessage(ev.ID, ev.Payload)
			msg.SetContext(requestid.WithContext(ctx, ev.CorrelationID))
			if err := pub.Publish(ev.Topic, msg); err != nil {
				return err
			}
			log.Info(msg.Context(), "outbox.published", "event_id", ev.ID, "topic", ev.Topic)
			return nil
		})
	})
}

func Poll(ctx context.Context, log *logger.Logger, name string, action func(context.Context) error) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		batchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := action(batchCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Error(ctx, name+".retry", "error", err.Error())
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
