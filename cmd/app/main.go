package main

import (
	"Flatly/internal/user"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// 1. Загружаем .env (если файл существует)
	_ = godotenv.Load() // не ошибка, если файла нет — значит переменные из окружения

	// 2. Читаем настройки HTTP
	host := getEnv("HTTP_HOST", "0.0.0.0")
	port := getEnv("HTTP_PORT", "8082")
	addr := fmt.Sprintf("%s:%s", host, port)

	readTimeout, _ := time.ParseDuration(getEnv("HTTP_READ_TIMEOUT", "5s"))
	writeTimeout, _ := time.ParseDuration(getEnv("HTTP_WRITE_TIMEOUT", "10s"))
	idleTimeout, _ := time.ParseDuration(getEnv("HTTP_IDLE_TIMEOUT", "60s"))

	// 3. Настраиваем пул PostgreSQL
	pgConfig, err := pgxpool.ParseConfig(getEnv("POSTGRES_DSN", ""))
	if err != nil {
		log.Fatalf("Невалидный POSTGRES_DSN: %v", err)
	}

	maxConns, _ := strconv.Atoi(getEnv("POSTGRES_MAX_CONNS", "20"))
	minConns, _ := strconv.Atoi(getEnv("POSTGRES_MIN_CONNS", "2"))
	maxLifetime, _ := time.ParseDuration(getEnv("POSTGRES_MAX_CONN_LIFETIME", "1h"))
	maxIdleTime, _ := time.ParseDuration(getEnv("POSTGRES_MAX_CONN_IDLE_TIME", "30m"))
	healthPeriod, _ := time.ParseDuration(getEnv("POSTGRES_HEALTH_CHECK_PERIOD", "1m"))

	pgConfig.MaxConns = int32(maxConns)
	pgConfig.MinConns = int32(minConns)
	pgConfig.MaxConnLifetime = maxLifetime
	pgConfig.MaxConnIdleTime = maxIdleTime
	pgConfig.HealthCheckPeriod = healthPeriod

	pool, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		log.Fatalf("Не удалось подключиться к PostgreSQL: %v", err)
	}
	defer pool.Close()

	log.Printf("PostgreSQL подключён (max: %d, min: %d)", pool.Config().MaxConns, pool.Config().MinConns)

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	repo := user.NewPgRepository(pool, logger)
	service := user.NewService(repo, logger)
	handler := user.NewHTTPHandler(service, logger)

	// 4. Создаём роутер (пример с chi)
	r := chi.NewRouter()

	r.Post("/users", handler.HandleCreateUser)

	// 5. Настраиваем и запускаем сервер
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	log.Printf("Сервер запускается на http://%s", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// Вспомогательная функция — безопасно читает переменную
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
