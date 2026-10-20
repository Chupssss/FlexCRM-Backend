package main

import (
	"context"
	"log"
	"net/http"
	"time"

	database "FlexCRM-Backend/db"
	"FlexCRM-Backend/internal/auth"
	"FlexCRM-Backend/internal/clients"
	"FlexCRM-Backend/internal/config"
	"FlexCRM-Backend/internal/health"
	"FlexCRM-Backend/internal/routers"
	"FlexCRM-Backend/internal/token"
	"FlexCRM-Backend/internal/users"

	"github.com/joho/godotenv"
)

func main() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Println("config: error loading .env file, use base variables")
	}

	cfg := config.Load()

	// HTTP сервер
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:              cfg.BaseURL,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Подключение к PostgreSQL
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	dbConnect, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed: ", err)
	}
	defer dbConnect.Close()

	log.Println("database connected")

	// JWT
	tokenManager := token.NewTokenManager(cfg.JWTSecret)

	// Health
	healthRepo := health.SaveRepoConn(dbConnect)
	healthService := health.NewService(healthRepo)
	healthHandler := health.NewHandler(healthService)

	// Clients
	clientsRepository := clients.NewRepository(dbConnect)
	clientsService := clients.NewService(clientsRepository)
	clientsHandler := clients.NewHandler(clientsService)

	// Auth
	authRepo := auth.SaveRepoConn(dbConnect)
	authService := auth.NewService(authRepo, tokenManager)
	authHandler := auth.NewHandler(authService)

	// Users
	usersRepository := users.NewRepository(dbConnect)
	usersService := users.NewService(usersRepository)
	usersHandler := users.NewHandler(usersService)

	// Регистрация маршрутов
	routers.SetupRouter(mux, routers.Dependencies{
		HealthHandler:  healthHandler,
		AuthHandler:    authHandler,
		ClientsHandler: clientsHandler,
		UsersHandler:   usersHandler,
		Tokenmanager:   tokenManager,
	})

	log.Println("server starting on ", cfg.BaseURL)

	// Запуск сервера
	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatal("server error: ", err)
	}
}
