package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go-project-template/service"
)

type UserOnboardingHandler struct {
	service service.UserOnboardingService
}

func NewUserOnboardingHandler(service service.UserOnboardingService) *UserOnboardingHandler {
	return &UserOnboardingHandler{service: service}
}

type startOnboardingRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type startOnboardingResponse struct {
	UserID             string `json:"user_id"`
	WorkflowInstanceID string `json:"workflow_instance_id"`
}

func (h *UserOnboardingHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req startOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, instanceID, err := h.service.Start(r.Context(), req.Name, req.Email, req.Address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(startOnboardingResponse{
		UserID:             userID,
		WorkflowInstanceID: instanceID,
	})
}

func (h *UserOnboardingHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	if err := h.service.VerifyEmail(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
