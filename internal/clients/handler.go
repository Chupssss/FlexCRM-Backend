package clients

import (
	"encoding/json"
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

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	// TODO: после готовности auth middleware
	// companyID брать из currentUser.CompanyID.

	companyID, err := uuid.Parse("11111111-1111-1111-1111-111111111111")
	if err != nil {
		http.Error(w, "invalid company id", http.StatusInternalServerError)
		return
	}

	clients, err := h.service.GetAll(r.Context(), companyID)
	if err != nil {
		log.Printf("clients: get all: %v", err)

		http.Error(
			w,
			"failed to get clients",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(clients); err != nil {
		log.Printf("clients: encode response: %v", err)
	}
}
