package main

import (
	"context"
	"log"
	"net/http"
	"time"

	database "FlexCRM-Backend/db"
	"FlexCRM-Backend/internal/config"
	"FlexCRM-Backend/internal/health"
	"FlexCRM-Backend/internal/routers"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("config: error loading .env file, use base variables")
	}

	cfg := config.Load()

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:              cfg.BaseURL,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db_connect, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed: ", err)
	}
	defer db_connect.Close()
	log.Println("database connected")

	// Инициализация health сервиса
	conn := health.SaveRepoConn(db_connect)           //Сохранение подключения к БД в структуре
	healthservice := health.NewService(conn)          // Инициализация mainservice
	healthhandler := health.NewHandler(healthservice) // Создание health хэндлера

	// Инициализация auths сервиса

	routers.SetupRouter(mux, routers.Dependencies{
		HealthHandler: healthhandler,
	}) // Инициализация эндпоинтов

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("server error: ", err)
	}
}
