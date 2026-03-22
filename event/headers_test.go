package event

import (
	"context"
	"testing"

	"go-project-template/requestid"
)

func TestHeaders_FromContextAndToMetadata(t *testing.T) {
	ctx := requestid.WithContext(context.Background(), "req-123")
	h := HeadersFromContext(ctx)
	if h.CorrelationID != "req-123" {
		t.Fatalf("expected correlation id from context, got %q", h.CorrelationID)
	}

	md := h.ToMetadata()
	if got := md.Get(MetadataCorrelationID); got != "req-123" {
		t.Fatalf("expected correlation id in metadata, got %q", got)
	}
}

func TestHeaders_FromMetadataAndInjectContext(t *testing.T) {
	md := map[string]string{MetadataCorrelationID: "req-xyz"}
	h := HeadersFromMetadata(md)
	if h.CorrelationID != "req-xyz" {
		t.Fatalf("expected correlation id from metadata, got %q", h.CorrelationID)
	}

	ctx := h.InjectContext(context.Background())
	if got := requestid.FromContext(ctx); got != "req-xyz" {
		t.Fatalf("expected request id in context, got %q", got)
	}
}
