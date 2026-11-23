package general

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"FlexCRM-Backend/internal/customerror"
	"FlexCRM-Backend/internal/middleware"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok || user.User_id == uuid.Nil || user.Company_id == uuid.Nil {
		customerror.SendError(w, &customerror.AppError{HTTPStatus: 401, Code: "UNAUTHORIZED", Message: "Требуется авторизация"})
		return
	}
	limit := 5
	if values, present := r.URL.Query()["limit"]; present {
		var err error
		if len(values) == 1 {
			limit, err = strconv.Atoi(values[0])
		}
		if len(values) != 1 || err != nil || limit < 1 || limit > 50 {
			customerror.SendError(w, &customerror.AppError{HTTPStatus: 400, Code: "INVALID_QUERY_PARAMETER", Message: "Параметр limit должен быть числом от 1 до 50"})
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	data, err := h.service.Get(ctx, user.User_id, user.Company_id, limit)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			customerror.SendError(w, &customerror.AppError{HTTPStatus: 403, Code: "FORBIDDEN", Message: "Доступ запрещён"})
		} else {
			log.Printf("general: %v", err)
			customerror.SendError(w, &customerror.AppError{HTTPStatus: 500, Code: "INTERNAL_ERROR", Message: "Не удалось получить данные главной страницы"})
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("general: encode response: %v", err)
	}
}
