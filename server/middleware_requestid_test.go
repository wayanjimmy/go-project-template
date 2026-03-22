package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-project-template/requestid"
)

func TestRequestIDMiddleware_UsesClientHeader(t *testing.T) {
	var gotCtxID string
	h := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtxID = requestid.FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestid.HeaderName, "from-client-1")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if gotCtxID != "from-client-1" {
		t.Fatalf("expected context request id from client, got %q", gotCtxID)
	}

	if got := rr.Header().Get(requestid.HeaderName); got != "from-client-1" {
		t.Fatalf("expected response header request id from client, got %q", got)
	}
}

func TestRequestIDMiddleware_GeneratesWhenMissing(t *testing.T) {
	var gotCtxID string
	h := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtxID = requestid.FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if !strings.HasPrefix(gotCtxID, requestid.GeneratedID) {
		t.Fatalf("expected generated context request id prefix %q, got %q", requestid.GeneratedID, gotCtxID)
	}

	if got := rr.Header().Get(requestid.HeaderName); got != gotCtxID {
		t.Fatalf("expected response header to match context id, got header=%q context=%q", got, gotCtxID)
	}
}
