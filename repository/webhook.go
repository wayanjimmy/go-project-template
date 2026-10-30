package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-project-template/logger"
	"go-project-template/internal/requestid"
	"go-project-template/service"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type WebhookRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewWebhookRepository(db *sqlx.DB, log *logger.Logger) *WebhookRepository {
	return &WebhookRepository{db, log}
}

func (r *WebhookRepository) Create(ctx context.Context, d service.WebhookDelivery) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO webhook_deliveries(id,event_id,destination,body,correlation_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT(event_id,destination) DO NOTHING`, d.ID, d.EventID, d.Destination, d.Body, d.CorrelationID)
	return err
}

func (r *WebhookRepository) Dispatch(ctx context.Context, queue service.WebhookQueue) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var items []service.WebhookDelivery
	if err := tx.SelectContext(ctx, &items, `SELECT id,event_id,destination,body,correlation_id,status,attempts FROM webhook_deliveries WHERE status='pending' AND next_attempt_at<=NOW() AND enqueue_after<=NOW() AND locked_until<=NOW() ORDER BY next_attempt_at LIMIT 50 FOR UPDATE SKIP LOCKED`); err != nil {
		return err
	}
	for _, d := range items {
		jobCtx := requestid.WithContext(ctx, d.CorrelationID)
		if err := queue.Enqueue(jobCtx, d.ID); err != nil {
			return err
		}
		// A durable backstop, not a permanent "enqueued" bit. Lost tasks are requeued.
		if _, err := tx.ExecContext(ctx, `UPDATE webhook_deliveries SET enqueue_after=NOW()+INTERVAL '2 minutes' WHERE id=$1`, d.ID); err != nil {
			return err
		}
		r.log.Info(jobCtx, "webhook.enqueued", "delivery_id", d.ID, "event_id", d.EventID)
	}
	return tx.Commit()
}

// Claim uses a durable lease so no database connection is held across HTTP.
func (r *WebhookRepository) Claim(ctx context.Context, id string) (*service.WebhookDelivery, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE webhook_deliveries SET status='failed',last_error='attempt limit reached after interrupted execution',finished_at=NOW() WHERE id=$1 AND status='pending' AND attempts >= $2 AND locked_until<=NOW()`, id, service.WebhookMaxAttempts)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n > 0 {
		r.log.Error(ctx, "webhook.failed", "delivery_id", id, "error", "attempt limit reached after interrupted execution")
	}
	var d service.WebhookDelivery
	err = r.db.GetContext(ctx, &d, `UPDATE webhook_deliveries SET attempts=attempts+1,locked_until=NOW()+INTERVAL '30 seconds',lease_token=$2 WHERE id=$1 AND status='pending' AND next_attempt_at<=NOW() AND locked_until<=NOW() AND attempts<$3 RETURNING id,event_id,destination,body,correlation_id,status,attempts,lease_token`, id, uuid.New().String(), service.WebhookMaxAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &d, err
}

func (r *WebhookRepository) Finish(ctx context.Context, d service.WebhookDelivery, status int, message, outcome string, delay time.Duration) error {
	result, err := r.db.ExecContext(ctx, `UPDATE webhook_deliveries SET status=$3,last_status=$4,last_error=$5,next_attempt_at=NOW()+$6*INTERVAL '1 second',locked_until='-infinity',finished_at=CASE WHEN $3='pending' THEN NULL ELSE NOW() END WHERE id=$1 AND lease_token=$2 AND status='pending'`, d.ID, d.LeaseToken, outcome, status, message, delay.Seconds())
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return fmt.Errorf("webhook delivery lease lost: %s", d.ID)
	}
	return nil
}
