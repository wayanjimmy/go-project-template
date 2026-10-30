package worker

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"go-project-template/event"
	"go-project-template/logger"
	"go-project-template/internal/publisher"
	"go-project-template/internal/requestid"
	"go-project-template/service"
	"time"
	"uuid"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hibiken/asynq"
)

func WebhookSubscriber(store service.WebhookStore, destination string, log *logger.Logger) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var user event.UserUpsertedEvent
		if err := json.Unmarshal(msg.Payload, &user); err != nil {
			return err
		}
		ctx := event.HeadersFromMetadata(msg.Metadata).InjectContext(msg.Context())
		body, err := json.Marshal(struct {
			ID        string `json:"id"`
			Type      string `json:"type"`
			CreatedAt string `json:"created_at"`
			Data      struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"data"`
		}{ID: msg.UUID, Type: event.UserCreated, CreatedAt: time.Now().UTC().Format(time.RFC3339), Data: struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{user.UserID, user.Name, user.Email}})
		if err != nil {
			return err
		}
		err = store.Create(ctx, service.WebhookDelivery{ID: uuid.New().String(), EventID: msg.UUID, Destination: destination, Body: body, CorrelationID: requestid.FromContext(ctx)})
		if err == nil {
			log.Info(ctx, "webhook.recorded", "event_id", msg.UUID, "user_id", user.UserID)
		}
		return err
	}
}

func NewAsynqServer(addr string, store service.WebhookStore, sender service.WebhookSender, retryBase time.Duration, log *logger.Logger) (*asynq.Server, *asynq.ServeMux) {
	delay := func(n int, _ error, _ *asynq.Task) time.Duration {
		// Redis retry timestamps have whole-second precision. Schedule after the
		// database's fractional deadline, never just before it.
		return min(retryBase*time.Duration(1<<min(n, 8)), time.Hour) + time.Second
	}
	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: addr}, asynq.Config{Concurrency: 4, Queues: map[string]int{"webhooks": 1}, ShutdownTimeout: 15 * time.Second, RetryDelayFunc: delay, DelayedTaskCheckInterval: time.Second, ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
		log.Error(ctx, "asynq.task_failed", "type", t.Type(), "error", err.Error())
	})})
	mux := asynq.NewServeMux()
	mux.HandleFunc(publisher.WebhookTask, func(ctx context.Context, t *asynq.Task) error {
		var p publisher.WebhookTaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("decode webhook task: %w: %w", err, asynq.SkipRetry)
		}
		if p.DeliveryID == "" {
			return fmt.Errorf("missing delivery ID: %w", asynq.SkipRetry)
		}
		err := service.DeliverWebhook(ctx, store, sender, p.DeliveryID, retryBase, log)
		if errors.Is(err, service.ErrWebhookFailed) {
			return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
		}
		return err
	})
	return srv, mux
}
