package requestid

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

const (
	HeaderName  = "X-Request-ID"
	GeneratedID = "gen_"
)

type contextKey struct{}

func WithContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// Resolve returns a request ID suitable for tracing.
// It prefers a client-provided header value; otherwise it generates
// a new ID with the GeneratedID prefix.
func Resolve(headerValue string) string {
	if id := strings.TrimSpace(headerValue); id != "" {
		return id
	}

	return GeneratedID + uuid.NewString()
}
