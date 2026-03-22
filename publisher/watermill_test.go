package publisher

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"go-project-template/event"
	"go-project-template/requestid"
)

type fakeMessagePublisher struct {
	lastTopic string
	lastMsg   *message.Message
}

func (f *fakeMessagePublisher) Publish(topic string, messages ...*message.Message) error {
	f.lastTopic = topic
	if len(messages) > 0 {
		f.lastMsg = messages[0]
	}
	return nil
}

func (f *fakeMessagePublisher) Close() error { return nil }

func decoratedPublisher(pub message.Publisher) *WatermillPublisher {
	decorated, _ := CorrelationIDDecorator()(pub)
	return NewWatermillPublisher(decorated)
}

func TestWatermillPublisher_Publish_PropagatesRequestIDFromContext(t *testing.T) {
	fake := &fakeMessagePublisher{}
	p := decoratedPublisher(fake)

	ctx := requestid.WithContext(context.Background(), "req-1")
	payload := map[string]string{"k": "v"}

	if err := p.Publish(ctx, "topic.a", payload); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if fake.lastMsg == nil {
		t.Fatal("expected message to be published")
	}

	if got := fake.lastMsg.Metadata.Get(event.MetadataCorrelationID); got != "req-1" {
		t.Fatalf("expected correlation id metadata req-1, got %q", got)
	}

	var gotPayload map[string]string
	if err := json.Unmarshal(fake.lastMsg.Payload, &gotPayload); err != nil {
		t.Fatalf("payload unmarshal failed: %v", err)
	}
	if gotPayload["k"] != "v" {
		t.Fatalf("unexpected payload: %+v", gotPayload)
	}
}

func TestWatermillPublisher_Publish_GeneratesRequestIDWhenMissing(t *testing.T) {
	fake := &fakeMessagePublisher{}
	p := decoratedPublisher(fake)

	if err := p.Publish(context.Background(), "topic.a", map[string]string{"ok": "1"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if fake.lastMsg == nil {
		t.Fatal("expected message to be published")
	}

	got := fake.lastMsg.Metadata.Get(event.MetadataCorrelationID)
	if got == "" {
		t.Fatal("expected generated request id in metadata")
	}
	if len(got) < len(requestid.GeneratedID) || got[:len(requestid.GeneratedID)] != requestid.GeneratedID {
		t.Fatalf("expected generated request id prefix %q, got %q", requestid.GeneratedID, got)
	}
}
