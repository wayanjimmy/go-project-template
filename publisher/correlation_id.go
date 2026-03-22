package publisher

import (
	"go-project-template/event"
	"go-project-template/requestid"

	"github.com/ThreeDotsLabs/watermill/message"
)

// CorrelationIDDecorator ensures each published message has a correlation ID.
//
// Priority:
//  1. Existing message metadata correlation ID (if already set)
//  2. Request ID from message context
//  3. Generated request ID
func CorrelationIDDecorator() message.PublisherDecorator {
	return func(pub message.Publisher) (message.Publisher, error) {
		return &correlationIDPublisher{wrapped: pub}, nil
	}
}

type correlationIDPublisher struct {
	wrapped message.Publisher
}

func (p *correlationIDPublisher) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		if msg.Metadata == nil {
			msg.Metadata = message.Metadata{}
		}

		if msg.Metadata.Get(event.MetadataCorrelationID) != "" {
			continue
		}

		correlationID := requestid.FromContext(msg.Context())
		if correlationID == "" {
			correlationID = requestid.Resolve("")
		}

		msg.Metadata.Set(event.MetadataCorrelationID, correlationID)
	}

	return p.wrapped.Publish(topic, messages...)
}

func (p *correlationIDPublisher) Close() error {
	return p.wrapped.Close()
}
