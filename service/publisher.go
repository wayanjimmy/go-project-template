package service

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}
