package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"go-project-template/apperror"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var requestValidator = newRequestValidator()

func newRequestValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	return v
}

func decodeJSONBody(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return apperror.InvalidArgument("invalid request body", []apperror.FieldDetail{{
			Field:   "body",
			Message: err.Error(),
		}})
	}

	// A second decode rejects valid JSON followed by trailing junk or extra values.
	var extra struct{}
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return apperror.InvalidArgument("invalid request body", []apperror.FieldDetail{{
			Field:   "body",
			Message: "request body must contain a single JSON object",
		}})
	}

	return apperror.InvalidArgument("invalid request body", []apperror.FieldDetail{{
		Field:   "body",
		Message: "request body must contain a single JSON object",
	}})
}

func validateRequest(req any) error {
	if err := requestValidator.Struct(req); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return apperror.Wrap(apperror.KindInvalidArgument, "request validation failed", err, nil)
		}

		details := make([]apperror.FieldDetail, 0, len(validationErrors))
		for _, validationError := range validationErrors {
			details = append(details, apperror.FieldDetail{
				Field:   validationError.Field(),
				Message: validationMessage(validationError),
			})
		}

		return apperror.InvalidArgument("request validation failed", details)
	}

	return nil
}

func validateUserID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return apperror.InvalidArgument("request validation failed", []apperror.FieldDetail{{
			Field:   "id",
			Message: "id must be a valid UUID",
		}})
	}

	return nil
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", err.Field())
	default:
		return fmt.Sprintf("%s is invalid", err.Field())
	}
}
