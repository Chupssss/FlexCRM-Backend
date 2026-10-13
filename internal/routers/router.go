package routers

import (
	"FlexCRM-Backend/internal/auth"
	"FlexCRM-Backend/internal/clients"
	"FlexCRM-Backend/internal/health"
	"FlexCRM-Backend/internal/middleware"
	"FlexCRM-Backend/internal/token"
	"net/http"
)

type Dependencies struct {
	HealthHandler  *health.Handler
	AuthHandler    *auth.Handler
	ClientsHandler *clients.Handler
	Tokenmanager   *token.TokenManager
}

func SetupRouter(mux *http.ServeMux, deps Dependencies) {
	// Пример вызова хэндлера с верификацией пользователя и сохранением данных в контексте
	//mux.Handle("GET /api/health", middleware.Auth(deps.Tokenmanager)(http.HandlerFunc(deps.HealthHandler.HealthCheck)))
	mux.HandleFunc("GET /api/health", http.HandlerFunc(deps.HealthHandler.HealthCheck))
	mux.HandleFunc("GET /api/health/{$}", http.HandlerFunc(deps.HealthHandler.HealthCheck))
	mux.HandleFunc("POST /api/register", deps.AuthHandler.Register)
	mux.HandleFunc("POST /api/login", deps.AuthHandler.Login)
	mux.HandleFunc("POST /api/logout", deps.AuthHandler.LogOut)
	mux.HandleFunc("POST /api/refresh", deps.AuthHandler.Refresh)
	mux.Handle("POST /api/register/user", middleware.Auth(deps.Tokenmanager)(http.HandlerFunc(deps.AuthHandler.RegisterUser)))

	h := deps.ClientsHandler
	routes := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"GET", "", h.GetAll},
		{"POST", "", h.Create},
		{"GET", "/{id}", h.GetByID},
		{"PUT", "/{id}", h.Update},
		{"DELETE", "/{id}", h.Delete},
		{"GET", "/{id}/contacts", h.GetContacts},
		{"POST", "/{id}/contacts", h.SaveContact},
		{"PUT", "/{id}/contacts/{contact_id}", h.SaveContact},
		{"DELETE", "/{id}/contacts/{contact_id}", h.DeleteContact},
		{"GET", "/{id}/comments", h.GetComments},
		{"POST", "/{id}/comments", h.SaveComment},
		{"PUT", "/{id}/comments/{comment_id}", h.SaveComment},
		{"DELETE", "/{id}/comments/{comment_id}", h.DeleteComment},
		{"GET", "/{id}/tags", h.GetClientTags},
		{"PUT", "/{id}/tags", h.SetTags},
	}
	for _, prefix := range []string{"/api/clients", "/api/v1/clients"} {
		for _, route := range routes {
			mux.HandleFunc(route.method+" "+prefix+route.path, route.handler)
			mux.HandleFunc(route.method+" "+prefix+route.path+"/{$}", route.handler)
		}
	}
	for _, prefix := range []string{"/api", "/api/v1"} {
		mux.HandleFunc("GET "+prefix+"/users", h.GetUsers)
		mux.HandleFunc("GET "+prefix+"/users/{$}", h.GetUsers)
		mux.HandleFunc("GET "+prefix+"/tags", h.GetTags)
		mux.HandleFunc("GET "+prefix+"/tags/{$}", h.GetTags)
		mux.HandleFunc("POST "+prefix+"/tags", h.CreateTag)
		mux.HandleFunc("POST "+prefix+"/tags/{$}", h.CreateTag)
	}
}
