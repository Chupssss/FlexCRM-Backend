package health

import (
	"net/http"
)

type Handler struct {
	healthservice *Service
}

// Создание хэндлера, содержащего сервисы
func NewHandler(healthserv *Service) *Handler {
	return &Handler{healthservice: healthserv}
}

// Функция проверки состояния сервера
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
