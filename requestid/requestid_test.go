package requestid

import (
	"context"
	"strings"
	"testing"
)

func TestResolve_UsesClientValueWhenValid(t *testing.T) {
	got := Resolve("client-123_ABC:xyz")
	if got != "client-123_ABC:xyz" {
		t.Fatalf("expected client value, got %q", got)
	}
}

func TestResolve_GeneratesWhenEmpty(t *testing.T) {
	got := Resolve("")
	if !strings.HasPrefix(got, GeneratedID) {
		t.Fatalf("expected generated id prefix %q, got %q", GeneratedID, got)
	}
}

func TestResolve_UsesNonEmptyValue(t *testing.T) {
	got := Resolve("any-value")
	if got != "any-value" {
		t.Fatalf("expected client value, got %q", got)
	}
}

func TestWithContextAndFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithContext(ctx, "abc-123")

	if got := FromContext(ctx); got != "abc-123" {
		t.Fatalf("expected request id in context, got %q", got)
	}
}
