package repository

import (
	"context"
	"encoding/json/v2"
	"go-project-template/requestid"
	"uuid"

	"github.com/jmoiron/sqlx"
)

func (r *PostgresUserRepository) RecordEvent(ctx context.Context, topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.exec.ExecContext(ctx, `INSERT INTO event_outbox(id,topic,payload,correlation_id) VALUES($1,$2,$3,$4)`, uuid.New().String(), topic, data, requestid.FromContext(ctx))
	return err
}

type OutboxEvent struct {
	ID            string `db:"id"`
	Topic         string `db:"topic"`
	Payload       []byte `db:"payload"`
	CorrelationID string `db:"correlation_id"`
}

// Relay holds one bounded batch until publication is acknowledged. A crash before
// commit republishes the same event IDs. Consumers must tolerate duplicates.
func RelayOutbox(ctx context.Context, db *sqlx.DB, publish func(OutboxEvent) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// One publisher at a time preserves commit-visible event order across replicas.
	var locked bool
	if err := tx.GetContext(ctx, &locked, `SELECT pg_try_advisory_xact_lock(78123401)`); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	var events []OutboxEvent
	if err := tx.SelectContext(ctx, &events, `SELECT id,topic,payload,correlation_id FROM event_outbox WHERE published_at IS NULL ORDER BY sequence LIMIT 50 FOR UPDATE`); err != nil {
		return err
	}
	for _, ev := range events {
		if err := publish(ev); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE event_outbox SET published_at=NOW() WHERE id=$1`, ev.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
