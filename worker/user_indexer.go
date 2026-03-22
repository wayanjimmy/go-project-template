package worker

import (
	"encoding/json"
	"go-project-template/event"
	"go-project-template/service"

	"github.com/ThreeDotsLabs/watermill/message"
)

type UserIndexerHandlers struct {
	indexer service.UserIndexer
}

func NewUserIndexerHandlers(indexer service.UserIndexer) *UserIndexerHandlers {
	return &UserIndexerHandlers{indexer: indexer}
}

func (h *UserIndexerHandlers) OnUserUpserted(msg *message.Message) error {
	var ev event.UserUpsertedEvent
	if err := json.Unmarshal(msg.Payload, &ev); err != nil {
		return err
	}

	headers := event.HeadersFromMetadata(msg.Metadata)
	ctx := headers.InjectContext(msg.Context())
	return h.indexer.Index(ctx, service.UserSearchDocument{
		UserID:   ev.UserID,
		Name:     ev.Name,
		Email:    ev.Email,
		Document: ev.Document,
	})
}

func (h *UserIndexerHandlers) OnUserDeleted(msg *message.Message) error {
	var ev event.UserDeletedEvent
	if err := json.Unmarshal(msg.Payload, &ev); err != nil {
		return err
	}

	headers := event.HeadersFromMetadata(msg.Metadata)
	ctx := headers.InjectContext(msg.Context())
	return h.indexer.DeleteIndex(ctx, ev.UserID)
}
