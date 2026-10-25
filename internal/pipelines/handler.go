package pipelines

import (
	"encoding/json"
	"net/http"

	"FlexCRM-Backend/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Get_Option(w http.ResponseWriter, r *http.Request) {
	companyID, ok := middleware.CompanyIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	options, err := h.service.Get_Option(r.Context(), companyID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}
