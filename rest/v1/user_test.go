package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-project-template/apperror"
	"go-project-template/entity"
	"go-project-template/logger"
	"go-project-template/requestid"

	"github.com/go-chi/chi/v5"
)

type fakeUserService struct {
	createFn   func(ctx context.Context, name, email, address string) (*entity.User, error)
	updateFn   func(ctx context.Context, id, name, email, address string) (*entity.User, error)
	deleteFn   func(ctx context.Context, id string) error
	findByIDFn func(ctx context.Context, id string) (*entity.User, error)
	listFn     func(ctx context.Context, limit, offset int) ([]entity.User, error)
}

func (f fakeUserService) Create(ctx context.Context, name, email, address string) (*entity.User, error) {
	if f.createFn != nil {
		return f.createFn(ctx, name, email, address)
	}
	return nil, nil
}

func (f fakeUserService) Update(ctx context.Context, id, name, email, address string) (*entity.User, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, id, name, email, address)
	}
	return nil, nil
}

func (f fakeUserService) Delete(ctx context.Context, id string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f fakeUserService) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, nil
}

func (f fakeUserService) List(ctx context.Context, limit, offset int) ([]entity.User, error) {
	if f.listFn != nil {
		return f.listFn(ctx, limit, offset)
	}
	return nil, nil
}

func TestUserHandlerCreateValidationError(t *testing.T) {
	handler := NewUserHandler(fakeUserService{}, logger.Noop())
	req := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(`{"name":"Jimmy"}`))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response errorEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != string(apperror.KindInvalidArgument) {
		t.Fatalf("expected error code %q, got %q", apperror.KindInvalidArgument, response.Error.Code)
	}

	details, ok := response.Error.Details.([]any)
	if !ok {
		t.Fatalf("expected validation details array, got %T", response.Error.Details)
	}

	fields := make(map[string]bool, len(details))
	for _, detail := range details {
		item, ok := detail.(map[string]any)
		if !ok {
			t.Fatalf("expected detail object, got %T", detail)
		}
		field, _ := item["field"].(string)
		fields[field] = true
	}

	if !fields["email"] || !fields["address"] {
		t.Fatalf("expected email and address validation errors, got %#v", fields)
	}
}

func TestUserHandlerGetNotFoundLogsStructuredError(t *testing.T) {
	var records []logger.Record
	log := logger.NewWithEvents(io.Discard, logger.LevelInfo, "test", requestid.FromContext, logger.Events{
		Error: func(ctx context.Context, record logger.Record) {
			records = append(records, record)
		},
	})

	handler := NewUserHandler(fakeUserService{
		findByIDFn: func(ctx context.Context, id string) (*entity.User, error) {
			return nil, fmt.Errorf("lookup failed: %w", apperror.NotFound("user not found", map[string]any{"resource": "user", "id": id}))
		},
	}, log)

	req := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000000", nil)
	req = req.WithContext(requestid.WithContext(req.Context(), "req-123"))
	req = withUserIDRouteParam(req, "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var response errorEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != string(apperror.KindNotFound) {
		t.Fatalf("expected error code %q, got %q", apperror.KindNotFound, response.Error.Code)
	}

	if len(records) == 0 {
		t.Fatal("expected an error log record")
	}

	record := records[len(records)-1]
	if record.Message != "rest.error_response" {
		t.Fatalf("expected log message rest.error_response, got %q", record.Message)
	}
	if got := record.Attributes["correlation_id"]; got != "req-123" {
		t.Fatalf("expected correlation_id req-123, got %#v", got)
	}
	if got := fmt.Sprint(record.Attributes["http_status"]); got != fmt.Sprint(http.StatusNotFound) {
		t.Fatalf("expected http_status %d, got %#v", http.StatusNotFound, record.Attributes["http_status"])
	}
	if got := record.Attributes["error_code"]; got != string(apperror.KindNotFound) {
		t.Fatalf("expected error_code %q, got %#v", apperror.KindNotFound, got)
	}
}

func TestUserHandlerGetInternalErrorHidesServerDetails(t *testing.T) {
	handler := NewUserHandler(fakeUserService{
		findByIDFn: func(ctx context.Context, id string) (*entity.User, error) {
			return nil, errors.New("database is down")
		},
	}, logger.Noop())

	req := httptest.NewRequest(http.MethodGet, "/v1/users/00000000-0000-0000-0000-000000000000", nil)
	req = withUserIDRouteParam(req, "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var response errorEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != string(apperror.KindInternal) {
		t.Fatalf("expected error code %q, got %q", apperror.KindInternal, response.Error.Code)
	}
	if response.Error.Message != "internal server error" {
		t.Fatalf("expected generic internal error message, got %q", response.Error.Message)
	}
}

func withUserIDRouteParam(req *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
