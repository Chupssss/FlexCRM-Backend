package routers

import (
	"net/http"

	"FlexCRM-Backend/internal/auth"
	"FlexCRM-Backend/internal/health"
	"FlexCRM-Backend/internal/middleware"
	"FlexCRM-Backend/internal/token"
	"FlexCRM-Backend/internal/users"
)

type Dependencies struct {
	HealthHandler *health.Handler
	AuthHandler   *auth.Handler
	UsersHandler  *users.Handler
	Tokenmanager  *token.TokenManager
}

func SetupRouter(mux *http.ServeMux, deps Dependencies) {
	// Health
	mux.HandleFunc("GET /api/health", deps.HealthHandler.HealthCheck)

	// Auth
	mux.HandleFunc("POST /api/register", deps.AuthHandler.Register)

	// Users (JWT authorization)
	mux.Handle(
		"GET /api/users",
		middleware.Auth(deps.Tokenmanager)(
			http.HandlerFunc(deps.UsersHandler.GetAll),
		),
	)

	mux.Handle(
		"GET /api/users/{id}",
		middleware.Auth(deps.Tokenmanager)(
			http.HandlerFunc(deps.UsersHandler.GetById),
		),
	)

	mux.Handle(
		"POST /api/users",
		middleware.Auth(deps.Tokenmanager)(
			http.HandlerFunc(deps.UsersHandler.Create),
		),
	)

	mux.Handle(
		"PATCH /api/users/{id}",
		middleware.Auth(deps.Tokenmanager)(
			http.HandlerFunc(deps.UsersHandler.Update),
		),
	)
}
