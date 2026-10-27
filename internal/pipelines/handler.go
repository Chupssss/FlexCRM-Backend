package pipelines

import (
	"encoding/json"
	"net/http"

	"FlexCRM-Backend/internal/middleware"

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

func (h *Handler) Get_Stages(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}
	companyID, ok := middleware.CompanyIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	stages, err := h.service.Get_Stages(r.Context(), companyID, pipelineID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stages)
}
