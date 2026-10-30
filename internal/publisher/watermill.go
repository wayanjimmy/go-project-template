package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type WatermillPublisher struct {
	pub message.Publisher
}

func NewWatermillPublisher(pub message.Publisher) *WatermillPublisher {
	return &WatermillPublisher{pub: pub}
}

func (w *WatermillPublisher) Publish(ctx context.Context, topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	msg.SetContext(context.WithoutCancel(ctx))

	return w.pub.Publish(topic, msg)
}
