package routers

import (
	"FlexCRM-Backend/internal/health"
	"net/http"
)

type Dependencies struct {
	HealthHandler *health.Handler
}

func SetupRouter(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /api/health/", deps.HealthHandler.HealthCheck)
	//mux.HandleFunc("/api/register/")
	// mux.HandleFunc("/api/login/")
	// mux.HandleFunc("/api/logou/")
	// mux.HandleFunc("/api/refresh/")
}
