package v1

import (
	"encoding/json"
	"go-project-template/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go-project-template/entity"
	"go-project-template/service"
)

type UserHandler struct {
	service service.UserService
	log     *logger.Logger
}

func NewUserHandler(service service.UserService, log *logger.Logger) *UserHandler {
	if log == nil {
		log = logger.Noop()
	}

	return &UserHandler{service: service, log: log}
}

type createUserRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type updateUserRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type userResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

func toUserResponse(user *entity.User) userResponse {
	return userResponse{
		ID:      user.ID,
		Name:    user.Name,
		Email:   user.Email,
		Address: user.Address,
	}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	h.log.Info(r.Context(), "handler.user.create")

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.Create(r.Context(), req.Name, req.Email, req.Address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.log.Info(r.Context(), "handler.user.update", "user_id", id)

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.Update(r.Context(), id, req.Name, req.Email, req.Address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.log.Info(r.Context(), "handler.user.get", "user_id", id)

	user, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.log.Info(r.Context(), "handler.user.delete", "user_id", id)

	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
