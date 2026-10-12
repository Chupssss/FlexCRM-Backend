package customerror

import (
	"encoding/json"
	"net/http"
)

// Пакет с созданием кастомной ошибки и возвратом ее на фронтэнд

// Структура хранящая данные ошибки
type AppError struct {
	HTTPStatus int    `json:"-"`       // не используем в json, так как передаем как header
	Code       string `json:"code"`    // Код ошибки для фронтенда
	Message    string `json:"message"` // Описание ошибки
}

// Реализация интерфейса error
func (e *AppError) Error() string {
	return e.Message
}

// Функция отправки ответа
func SendError(w http.ResponseWriter, appErr *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)

	json.NewEncoder(w).Encode(appErr)
}
