package middleware

import (
	"FlexCRM-Backend/internal/customerror"
	"FlexCRM-Backend/internal/token"
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "current_user"

// Структура содержащая данные о пользователе, которые помещаются в контекст
type CurrentUser struct {
	User_id    uuid.UUID
	Company_id uuid.UUID
	Role       string
}

// Middleware функция для проверки залогинин ли пользователь
func Auth(manager *token.TokenManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			access_token, err := r.Cookie("access-token")
			if err != nil {
				log.Printf("%v", err)
				if errors.Is(err, http.ErrNoCookie) {
					customerror.SendError(w, &customerror.AppError{
						HTTPStatus: http.StatusUnauthorized,
						Code:       "UNAUTHORIZED",
						Message:    "Требуется авторизация",
					})
					return
				}
				customerror.SendError(w, &customerror.AppError{
					HTTPStatus: http.StatusInternalServerError,
					Code:       "INTERNAL_ERROR",
					Message:    "Не удалось получить Cookie",
				})
				return
			}
			claims, err := manager.VerifyToken(access_token.Value)
			if err != nil {
				customerror.SendError(w, &customerror.AppError{
					HTTPStatus: http.StatusUnauthorized,
					Code:       "UNAUTHORIZED",
					Message:    "Требуется авторизация",
				})
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, CurrentUser{
				User_id:    claims.User_id,
				Company_id: claims.Company_id,
				Role:       claims.Role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Функция, которая достает данные пользователя из контекста
func UserFromContext(ctx context.Context) (CurrentUser, bool) {
	user, ok := ctx.Value(userContextKey).(CurrentUser)
	return user, ok
}
