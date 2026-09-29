package auth

import (
	"FlexCRM-Backend/internal/customerror"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	service *Service
}

// Функция сохранения сервиса в хэндлер
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Структура содержащая данные о компании и администраторе
type admin_company struct {
	Company_name string `json:"company_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	First_name   string `json:"first_name"`
	Last_name    string `json:"last_name"`
}

// Функция регистрации администратора и компании
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apperr := customerror.AppError{HTTPStatus: 405, Code: "METHOD_NOT_ALLOWED", Message: "Неверный метод запроса"}
		customerror.SendError(w, &apperr)
		log.Println("error: method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // контекст, если пользователь закроет соединение или мы будет действовать более 5 секунд
	defer cancel()

	var user admin_company
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		apperr := customerror.AppError{HTTPStatus: 400, Code: "INVALID_REQUEST_BODY", Message: "Некорректное тело запроса"}
		customerror.SendError(w, &apperr)
		log.Println("%v", err)
		return
	}

	if user.Company_name == "" || user.Email == "" || user.First_name == "" || user.Last_name == "" || user.Password == "" {
		apperr := customerror.AppError{HTTPStatus: 400, Code: "VALIDATION_ERROR", Message: "Проверьте название компании и данные администратора"}
		customerror.SendError(w, &apperr)
		log.Println("error: invalid data")
		return
	}
	result, err := h.service.Register(ctx, &user)
	// обработка ошибки
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailAlreadyExists):
			customerror.SendError(w, &customerror.AppError{Code: "EMAIL_ALREADY_EXISTS",
				Message:    "Пользователь с таким email уже зарегистрирован",
				HTTPStatus: 409,
			})
			return
		default:
			apperr := customerror.AppError{HTTPStatus: 500, Code: "INTERNAL_ERROR", Message: "Не удалось зарегистрировать компанию и администратора"}
			customerror.SendError(w, &apperr)
		}
	}
}
