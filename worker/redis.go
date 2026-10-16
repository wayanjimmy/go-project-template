package worker

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
)

const (
	// consumerGroup is shared by all worker replicas so each event is
	// delivered to exactly one worker in the group.
	consumerGroup = "app-worker"

	topicUserCreated = "user_created"
	topicUserUpdated = "user_updated"
	topicUserDeleted = "user_deleted"
)

func newRedisStreamSubscriber(ctx context.Context, rdb *redis.Client, logger watermill.LoggerAdapter) (*redisstream.Subscriber, error) {
	if rdb == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: consumerGroup,
	}, logger)
}
