package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"Flatly/internal/auth"
	"Flatly/internal/flat"
	"Flatly/internal/house"
	"Flatly/internal/notify"
	"Flatly/pkg/config"
	httprouter "Flatly/pkg/httproute"
	"Flatly/pkg/postgres"
	"Flatly/pkg/sender"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	pool, err := postgres.NewPool(context.Background(), cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres init error: %v", err)
	}
	defer pool.Close()

	tokenManager := auth.NewHMACTokenManager(cfg.Auth.TokenSecret)
	authRepo := auth.NewPGRepository(pool)
	authService := auth.NewService(authRepo, tokenManager)
	authHandler := auth.NewHTTPHandler(authService)

	houseRepo := house.NewPGRepository(pool)
	flatRepo := flat.NewPGRepository(pool)
	notifyDispatcher := notify.NewDispatcher(houseRepo, sender.New())

	houseService := house.NewService(houseRepo, flatRepo)
	houseHandler := house.NewHTTPHandler(houseService)

	flatService := flat.NewService(flatRepo, notifyDispatcher)
	flatHandler := flat.NewHTTPHandler(flatService)

	router := httprouter.NewRouter(authHandler, houseHandler, flatHandler, authService)

	addr := fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	log.Printf("Server started at http://%s", addr)
	log.Fatal(server.ListenAndServe())
}
