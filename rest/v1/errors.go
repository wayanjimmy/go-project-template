package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"go-project-template/apperror"
	"go-project-template/logger"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func respondJSON(log *logger.Logger, ctx context.Context, w http.ResponseWriter, status int, payload any) {
	if log == nil {
		log = logger.Noop()
	}
	if ctx == nil {
		ctx = context.Background()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(ctx, "rest.response_encode_failed", "http_status", status, "error", err.Error())
	}
}

func respondError(log *logger.Logger, w http.ResponseWriter, r *http.Request, err error) {
	if log == nil {
		log = logger.Noop()
	}

	status, response := classifyError(err)
	logArgs := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"http_status", status,
		"error_code", response.Error.Code,
		"error", err.Error(),
	}
	if response.Error.Details != nil {
		logArgs = append(logArgs, "details", response.Error.Details)
	}

	log.Error(r.Context(), "rest.error_response", logArgs...)
	respondJSON(log, r.Context(), w, status, response)
}

func classifyError(err error) (int, errorEnvelope) {
	appErr := apperror.As(err)
	if appErr == nil {
		return http.StatusInternalServerError, errorEnvelope{Error: apiError{
			Code:    string(apperror.KindInternal),
			Message: "internal server error",
		}}
	}

	status := http.StatusInternalServerError
	message := appErr.Message
	details := appErr.Details

	switch appErr.Kind {
	case apperror.KindInvalidArgument:
		status = http.StatusBadRequest
		if message == "" {
			message = "request validation failed"
		}
	case apperror.KindNotFound:
		status = http.StatusNotFound
		if message == "" {
			message = "resource not found"
		}
	case apperror.KindConflict:
		status = http.StatusConflict
		if message == "" {
			message = "conflict"
		}
	case apperror.KindUnauthorized:
		status = http.StatusUnauthorized
		if message == "" {
			message = "unauthorized"
		}
	case apperror.KindForbidden:
		status = http.StatusForbidden
		if message == "" {
			message = "forbidden"
		}
	default:
		status = http.StatusInternalServerError
		message = "internal server error"
		details = nil
	}

	return status, errorEnvelope{Error: apiError{
		Code:    string(appErr.Kind),
		Message: message,
		Details: details,
	}}
}
