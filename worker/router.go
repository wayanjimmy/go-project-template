package worker

import (
	"fmt"
	"go-project-template/event"
	"go-project-template/service"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewRouter(sub message.Subscriber, indexer service.UserIndexer, logger watermill.LoggerAdapter) (*message.Router, error) {
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
