package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-project-template/entity"
	"go-project-template/service"
)

type SearchHandler struct {
	service service.SearchService
}

func NewSearchHandler(service service.SearchService) *SearchHandler {
	return &SearchHandler{service: service}
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
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}

	items, err := h.service.Users(r.Context(), query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toSearchUsersResponse(items))
}
