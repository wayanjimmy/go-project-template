package v1

import (
	"errors"
	"net/http"

	"go-project-template/logger"
)

type ErrorExamplesHandler struct {
	log *logger.Logger
}

func NewErrorExamplesHandler(log *logger.Logger) *ErrorExamplesHandler {
	if log == nil {
		log = logger.Noop()
	}

	return &ErrorExamplesHandler{log: log}
}

// Internal demonstrates the standardized internal-error envelope for HTTP clients.
func (h *ErrorExamplesHandler) Internal(w http.ResponseWriter, r *http.Request) {
	respondError(h.log, w, r, errors.New("forced internal error example"))
}
