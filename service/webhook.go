package service

import (
	"context"
	"errors"
	"fmt"
	"go-project-template/logger"
	"go-project-template/requestid"
	"time"
)

type WebhookDelivery struct {
	ID            string `db:"id"`
	EventID       string `db:"event_id"`
	Destination   string `db:"destination"`
	Body          []byte `db:"body"`
	CorrelationID string `db:"correlation_id"`
	Status        string `db:"status"`
	Attempts      int    `db:"attempts"`
	LeaseToken    string `db:"lease_token"`
}

type WebhookQueue interface {
	Enqueue(ctx context.Context, deliveryID string) error
}

type WebhookSender interface {
	Send(ctx context.Context, destination, deliveryID string, body []byte) (int, error)
}

type WebhookStore interface {
	Create(ctx context.Context, delivery WebhookDelivery) error
	Dispatch(ctx context.Context, queue WebhookQueue) error
	Claim(ctx context.Context, id string) (*WebhookDelivery, error)
	Finish(ctx context.Context, d WebhookDelivery, status int, message, outcome string, delay time.Duration) error
}

const WebhookMaxAttempts = 6

var ErrWebhookFailed = errors.New("webhook delivery permanently failed")

func DeliverWebhook(ctx context.Context, store WebhookStore, sender WebhookSender, id string, retryBase time.Duration, log *logger.Logger) error {
	d, err := store.Claim(ctx, id)
	if err != nil || d == nil {
		return err
	}
	ctx = requestid.WithContext(ctx, d.CorrelationID)
	status, sendErr := sender.Send(ctx, d.Destination, d.ID, d.Body)
	outcome := "pending"
	message := ""
	if sendErr != nil {
		message = sendErr.Error()
	} else if status < 200 || status >= 300 {
		message = fmt.Sprintf("HTTP %d", status)
	}
	switch {
	case sendErr == nil && status >= 200 && status < 300:
		outcome = "succeeded"
	case d.Attempts >= WebhookMaxAttempts:
		outcome = "failed"
	case sendErr == nil && status != 408 && status != 429 && status < 500:
		outcome = "failed"
	}
	delay := min(retryBase*time.Duration(1<<min(d.Attempts-1, 8)), time.Hour)
	if err := store.Finish(ctx, *d, status, message, outcome, delay); err != nil {
		return err
	}
	args := []any{"delivery_id", d.ID, "event_id", d.EventID, "attempt", d.Attempts, "http_status", status, "outcome", outcome}
	if outcome == "failed" {
		log.Error(ctx, "webhook.failed", append(args, "error", message)...)
		return ErrWebhookFailed
	}
	log.Info(ctx, "webhook.attempt", args...)
	if outcome == "pending" {
		return fmt.Errorf("webhook delivery retry: %s", message)
	}
	return nil
}
