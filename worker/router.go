package worker

import (
	"context"
	"fmt"

	"go-project-template/event"
	"go-project-template/service"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

// NewRouter builds the worker's message router.
//
// Exactly one subscriber source must be provided:
//   - Distributed (production, multi-process): pass a *redis.Client to
//     subscribe via Redis Streams consumer group "app-worker".
//   - In-process (development, single binary): pass a gochannel subscriber.
func NewRouter(ctx context.Context, rdb *redis.Client, inProcessSub message.Subscriber, indexer service.UserIndexer, logger watermill.LoggerAdapter) (*message.Router, error) {
	if indexer == nil {
		return nil, fmt.Errorf("indexer is required")
	}

	var sub message.Subscriber
	switch {
	case rdb != nil:
		s, err := newRedisStreamSubscriber(ctx, rdb, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create redis stream subscriber: %w", err)
		}
		sub = s
	case inProcessSub != nil:
		sub = inProcessSub
	default:
		return nil, fmt.Errorf("either a redis client or an in-process subscriber is required")
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create watermill router: %w", err)
	}

		handlers := NewUserIndexerHandlers(indexer)

	router.AddNoPublisherHandler(
		"index-user-created",
		event.UserCreated,
		sub,
		handlers.OnUserUpserted,
	)

	router.AddNoPublisherHandler(
		"index-user-updated",
		event.UserUpdated,
		sub,
		handlers.OnUserUpserted,
	)

	router.AddNoPublisherHandler(
		"index-user-deleted",
		event.UserDeleted,
		sub,
		handlers.OnUserDeleted,
	)

	return router, nil
}
