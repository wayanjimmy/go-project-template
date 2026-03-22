package web

import (
	"net/http"
	"strconv"

	"go-project-template/entity"
	"go-project-template/logger"
	"go-project-template/service"

	inertia "github.com/petaki/inertia-go"
)

type UsersPageHandler struct {
	inertia       *inertia.Inertia
	userService   service.UserService
	searchService service.SearchService
	log           *logger.Logger
}

func NewUsersPageHandler(i *inertia.Inertia, userService service.UserService, searchService service.SearchService, log *logger.Logger) *UsersPageHandler {
	if log == nil {
		log = logger.Noop()
	}
	return &UsersPageHandler{
		inertia:       i,
		userService:   userService,
		searchService: searchService,
		log:           log,
	}
}

func (h *UsersPageHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 20
	offset := 0

	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			offset = v
		}
	}

	var (
		users []entity.User
		err   error
		mode  = "list"
		props = make(map[string]any)
	)

	if q != "" {
		mode = "search"
		users, err = h.searchService.Users(r.Context(), q, limit)
	} else {
		// Fetch one extra to check for "has more"
		items, listErr := h.userService.List(r.Context(), limit+1, offset)
		if listErr == nil {
			hasMore := len(items) > limit
			if hasMore {
				items = items[:limit]
			}
			users = items
			props = map[string]any{
				"has_more": hasMore,
			}
		}
		err = listErr
	}
	if users == nil {
		users = []entity.User{}
	}
	if err != nil {
		h.log.Error(r.Context(), "web.users.index", "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if props == nil {
		props = make(map[string]any)
	}
	props["users"] = users
	props["query"] = q
	props["mode"] = mode
	props["limit"] = limit
	props["offset"] = offset

	if err := h.inertia.Render(w, r, "UsersIndex", props); err != nil {
		h.log.Error(r.Context(), "web.users.render", "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
