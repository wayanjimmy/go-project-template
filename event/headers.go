package event

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	wmMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go-project-template/requestid"
)

const MetadataCorrelationID = wmMiddleware.CorrelationIDMetadataKey

type Headers struct {
	CorrelationID string
}

func HeadersFromContext(ctx context.Context) Headers {
	return Headers{CorrelationID: requestid.FromContext(ctx)}
}

func HeadersFromMetadata(metadata message.Metadata) Headers {
	if metadata == nil {
		return Headers{}
	}

	return Headers{CorrelationID: metadata.Get(MetadataCorrelationID)}
}

func (h Headers) ToMetadata() message.Metadata {
	metadata := message.Metadata{}
	if h.CorrelationID != "" {
		metadata.Set(MetadataCorrelationID, h.CorrelationID)
	}
	return metadata
}

func (h Headers) InjectContext(ctx context.Context) context.Context {
	if h.CorrelationID == "" {
		return ctx
	}

	return requestid.WithContext(ctx, h.CorrelationID)
}
