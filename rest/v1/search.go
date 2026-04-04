package v1

import (
	"go-project-template/apperror"
	"go-project-template/logger"
	"net/http"
	"strconv"

	"go-project-template/entity"
	"go-project-template/service"
)

type SearchHandler struct {
	service service.SearchService
	log     *logger.Logger
}

func NewSearchHandler(service service.SearchService, log *logger.Logger) *SearchHandler {
	if log == nil {
		log = logger.Noop()
	}

	return &SearchHandler{service: service, log: log}
}

type searchUsersResponseItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func toSearchUsersResponse(items []entity.User) []searchUsersResponseItem {
	out := make([]searchUsersResponseItem, 0, len(items))
	for _, it := range items {
		out = append(out, searchUsersResponseItem{
			ID:    it.ID,
			Name:  it.Name,
			Email: it.Email,
		})
	}
	return out
}

func (h *SearchHandler) Users(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			respondError(h.log, w, r, apperror.InvalidArgument("request validation failed", []apperror.FieldDetail{{
				Field:   "limit",
				Message: "limit must be a valid integer",
			}}))
			return
		}
		if v <= 0 {
			respondError(h.log, w, r, apperror.InvalidArgument("request validation failed", []apperror.FieldDetail{{
				Field:   "limit",
				Message: "limit must be greater than 0",
			}}))
			return
		}
		limit = v
	}

	items, err := h.service.Users(r.Context(), query, limit)
	if err != nil {
		respondError(h.log, w, r, err)
		return
	}

	respondJSON(h.log, r.Context(), w, http.StatusOK, toSearchUsersResponse(items))
}
