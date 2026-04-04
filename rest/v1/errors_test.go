package v1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-project-template/logger"
	"go-project-template/requestid"
)

func TestRespondJSONLogsEncodingFailure(t *testing.T) {
	var records []logger.Record
	log := logger.NewWithEvents(io.Discard, logger.LevelInfo, "test", requestid.FromContext, logger.Events{
		Error: func(ctx context.Context, record logger.Record) {
			records = append(records, record)
		},
	})

	rr := httptest.NewRecorder()
	ctx := requestid.WithContext(context.Background(), "req-encode")

	respondJSON(log, ctx, rr, http.StatusOK, map[string]any{
		"bad": func() {},
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if len(records) == 0 {
		t.Fatal("expected an error log record")
	}

	record := records[len(records)-1]
	if record.Message != "rest.response_encode_failed" {
		t.Fatalf("expected log message rest.response_encode_failed, got %q", record.Message)
	}
	if got := record.Attributes["correlation_id"]; got != "req-encode" {
		t.Fatalf("expected correlation_id req-encode, got %#v", got)
	}
	if got := fmt.Sprint(record.Attributes["http_status"]); got != fmt.Sprint(http.StatusOK) {
		t.Fatalf("expected http_status %d, got %#v", http.StatusOK, record.Attributes["http_status"])
	}
	if got := fmt.Sprint(record.Attributes["error"]); got == "" {
		t.Fatal("expected encode error text in log")
	}
}
