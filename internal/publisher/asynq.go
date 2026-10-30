package publisher

import (
	"context"
	"encoding/json/v2"
	"errors"
	"time"

	"github.com/hibiken/asynq"
)

const WebhookTask = "webhook:deliver"

type WebhookTaskPayload struct {
	DeliveryID string `json:"delivery_id"`
}

type AsynqQueue struct{ client *asynq.Client }

func NewAsynqQueue(addr string) *AsynqQueue {
	return &AsynqQueue{asynq.NewClient(asynq.RedisClientOpt{Addr: addr})}
}
func (q *AsynqQueue) Close() error { return q.client.Close() }
func (q *AsynqQueue) Enqueue(ctx context.Context, id string) error {
	data, err := json.Marshal(WebhookTaskPayload{DeliveryID: id})
	if err != nil {
		return err
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(WebhookTask, data), asynq.Queue("webhooks"), asynq.MaxRetry(5), asynq.Timeout(20*time.Second), asynq.Unique(time.Minute))
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}
