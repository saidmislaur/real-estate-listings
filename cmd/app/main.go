package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"Flatly/internal/user"
	"Flatly/pkg/config"
	httprouter "Flatly/pkg/httproute"
	"Flatly/pkg/postgres"
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

	repo := user.NewPgRepository(pool, logger)
	service := user.NewService(repo, logger)
	handler := user.NewHTTPHandler(service, logger)

	router := httprouter.NewRouter(handler, logger)

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
