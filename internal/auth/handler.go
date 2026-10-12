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
		log.Printf("error: method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // контекст, если пользователь закроет соединение или мы будет действовать более 5 секунд
	defer cancel()

	var user admin_company
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		apperr := customerror.AppError{HTTPStatus: 400, Code: "INVALID_REQUEST_BODY", Message: "Некорректное тело запроса"}
		customerror.SendError(w, &apperr)
		log.Printf("%v", err)
		return
	}

	if user.Company_name == "" || user.Email == "" || user.First_name == "" || user.Last_name == "" || user.Password == "" {
		apperr := customerror.AppError{HTTPStatus: 400, Code: "VALIDATION_ERROR", Message: "Проверьте название компании и данные администратора"}
		customerror.SendError(w, &apperr)
		log.Printf("error: invalid data")
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
			return
		}
	}
	h.setCookie(w, result) // Установка Cookie
	w.WriteHeader(http.StatusCreated)
}

// Структура хранящая данные юзера
type user struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Функция логина пользователя
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apperr := customerror.AppError{HTTPStatus: 405, Code: "METHOD_NOT_ALLOWED", Message: "Неверный метод запроса"}
		customerror.SendError(w, &apperr)
		log.Printf("error: method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // контекст, если пользователь закроет соединение или мы будет действовать более 5 секунд
	defer cancel()

	var us user
	err := json.NewDecoder(r.Body).Decode(&us)
	if err != nil || us.Email == "" || us.Password == "" {
		apperr := customerror.AppError{HTTPStatus: 400, Code: "INVALID_REQUEST_BODY", Message: "Некорректное тело запроса"}
		customerror.SendError(w, &apperr)
		log.Printf("%v", err)
		return
	}
	result, err := h.service.Login(ctx, &us)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			customerror.SendError(w, &customerror.AppError{Code: "INVALID_CREDENTIALS",
				Message:    "Неверный email или пароль",
				HTTPStatus: 401,
			})
			return
		case errors.Is(err, ErrDifferentPass):
			customerror.SendError(w, &customerror.AppError{Code: "INVALID_CREDENTIALS",
				Message:    "Неверный email или пароль",
				HTTPStatus: 401,
			})
			return
		case errors.Is(err, ErrUserDeactive):
			customerror.SendError(w, &customerror.AppError{Code: "USER_INACTIVE",
				Message:    "Пользователь деактивирован",
				HTTPStatus: 403,
			})
			return
		default:
			apperr := customerror.AppError{HTTPStatus: 500, Code: "INTERNAL_ERROR", Message: "Не удалось выполнить вход"}
			customerror.SendError(w, &apperr)
			return
		}
	}
	h.setCookie(w, result) // установка Cookie
	w.WriteHeader(http.StatusOK)
}

// Функция установки Cookie в Response
func (h *Handler) setCookie(w http.ResponseWriter, tokens *AuthResult) {
	// Установка access token в Cookie. Потом заменить secure на true, после появления https
	http.SetCookie(w, &http.Cookie{
		Name:     "access-token",
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  tokens.AccessExpiresAt,
	})
	// Установка refresh token в Cookie. Потом заменить Secure на true, после появления https
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh-token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  tokens.RefreshExpiresAt,
	})
}

// Функция обновления Refresh и access tokens
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apperr := customerror.AppError{HTTPStatus: 405, Code: "METHOD_NOT_ALLOWED", Message: "Неверный метод запроса"}
		customerror.SendError(w, &apperr)
		log.Printf("error: method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // контекст, если пользователь закроет соединение или мы будет действовать более 5 секунд
	defer cancel()

	cookie, err := r.Cookie("refresh-token")
	if err != nil {
		log.Printf("error refresh cookie: %v", err)
		customerror.SendError(w, &customerror.AppError{Code: "INVALID_REFRESH_TOKEN",
			Message:    "Необходимо повторно войти в систему",
			HTTPStatus: 401,
		})
		return
	}
	result, err := h.service.RefreshTokens(ctx, cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, ErrRefreshTokenNotFound):
			customerror.SendError(w, &customerror.AppError{Code: "INVALID_REFRESH_TOKEN",
				Message:    "Необходимо повторно войти в систему",
				HTTPStatus: 401,
			})
			return
		case errors.Is(err, ErrRefreshTokenExpired):
			customerror.SendError(w, &customerror.AppError{Code: "INVALID_REFRESH_TOKEN",
				Message:    "Необходимо повторно войти в систему",
				HTTPStatus: 401,
			})
			return
		case errors.Is(err, ErrUserNotFound):
			customerror.SendError(w, &customerror.AppError{Code: "INVALID_REFRESH_TOKEN",
				Message:    "Необходимо повторно войти в систему",
				HTTPStatus: 401,
			})
			return
		default:
			apperr := customerror.AppError{HTTPStatus: 500, Code: "INTERNAL_ERROR", Message: "Не удалось обновить токены"}
			customerror.SendError(w, &apperr)
			return
		}
	}
	h.setCookie(w, result)
	w.WriteHeader(http.StatusNoContent)
}

// Функция LogOut пользователя
func (h *Handler) LogOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apperr := customerror.AppError{HTTPStatus: 405, Code: "METHOD_NOT_ALLOWED", Message: "Неверный метод запроса"}
		customerror.SendError(w, &apperr)
		log.Printf("error: method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // контекст, если пользователь закроет соединение или мы будет действовать более 5 секунд
	defer cancel()

	cookie, err := r.Cookie("refresh-token")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "access-token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		w.WriteHeader(http.StatusNoContent)
		return
	}
	err = h.service.LogOut(ctx, cookie.Value)
	if err != nil {
		if !errors.Is(err, ErrRefreshTokenNotFound) {
			apperr := customerror.AppError{HTTPStatus: 500, Code: "INTERNAL_ERROR", Message: "Не удалось удалить токены"}
			customerror.SendError(w, &apperr)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "access-token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh-token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}
