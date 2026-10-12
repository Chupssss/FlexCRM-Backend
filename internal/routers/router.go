package routers

import (
	"FlexCRM-Backend/internal/auth"
	"FlexCRM-Backend/internal/health"
	"FlexCRM-Backend/internal/token"
	"net/http"
)

type Dependencies struct {
	HealthHandler *health.Handler
	AuthHandler   *auth.Handler
	Tokenmanager  *token.TokenManager
}

func SetupRouter(mux *http.ServeMux, deps Dependencies) {
	// Пример вызова хэндлера с верификацией пользователя и сохранением данных в контексте
	//	mux.Handle("GET /api/health", middleware.Auth(deps.Tokenmanager)(http.HandlerFunc(deps.HealthHandler.HealthCheck)))
	mux.HandleFunc("GET /api/health", http.HandlerFunc(deps.HealthHandler.HealthCheck))
	mux.HandleFunc("POST /api/register", deps.AuthHandler.Register)
	mux.HandleFunc("POST /api/login", deps.AuthHandler.Login)
	// mux.HandleFunc("/api/logout")
	mux.HandleFunc("POST /api/refresh", deps.AuthHandler.Refresh)
}
