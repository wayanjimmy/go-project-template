package apperror

import "errors"

type Kind string

const (
	KindInvalidArgument Kind = "invalid_argument"
	KindNotFound        Kind = "not_found"
	KindConflict        Kind = "conflict"
	KindUnauthorized    Kind = "unauthorized"
	KindForbidden       Kind = "forbidden"
	KindInternal        Kind = "internal"
)

type FieldDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Error struct {
	Kind    Kind
	Message string
	Details any
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message == "" {
		if e.Err != nil {
			return e.Err.Error()
		}
		return string(e.Kind)
	}

	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}

	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(kind Kind, message string, details any) *Error {
	return &Error{Kind: kind, Message: message, Details: details}
}

func Wrap(kind Kind, message string, err error, details any) *Error {
	return &Error{Kind: kind, Message: message, Details: details, Err: err}
}

func InvalidArgument(message string, details any) *Error {
	return New(KindInvalidArgument, message, details)
}

func NotFound(message string, details any) *Error {
	return New(KindNotFound, message, details)
}

func As(err error) *Error {
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return nil
}
