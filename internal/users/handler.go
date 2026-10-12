package users

import (
	"FlexCRM-Backend/internal/middleware"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из JWT-контекста
	currentUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Только ADMIN может создавать сотрудников
	if currentUser.Role != "ADMIN" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Передаём companyID из JWT в Service
	createdUser, err := h.service.Create(
		r.Context(),
		currentUser.Company_id,
		req,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidUserData) {
			http.Error(w, "invalid user data", http.StatusBadRequest)
			return
		}

		log.Printf("users.Create error: %v", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdUser); err != nil {
		log.Printf("failed to encode user: %v", err)
	}
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if currentUser.Role != "ADMIN" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	updatedUser, err := h.service.Update(
		r.Context(),
		userID,
		currentUser.Company_id,
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			http.Error(w, "user not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidUserData):
			http.Error(w, "invalid user data", http.StatusBadRequest)
		default:
			http.Error(w, "failed to update user", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
		log.Printf("failed to encode user: %v", err)
	}
}
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	users, err := h.service.GetAll(
		r.Context(), currentUser.Company_id,
	)
	if err != nil {
		http.Error(w, "failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Printf("failed to encode users response: %v", err)
		return
	}
}
func (h *Handler) GetById(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	user, err := h.service.GetById(
		r.Context(),
		id,
		currentUser.Company_id,
	)
	if err != nil {
		http.Error(w, "faled to get user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(
			w, "faled to enocode response",
			http.StatusInternalServerError,
		)
		return

	}

}
