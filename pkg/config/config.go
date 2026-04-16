package config

import (
	"os"
	"strconv"
	"time"
)

type HTTPConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type PostgresConfig struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
}

type AuthConfig struct {
	TokenSecret string
}

func Load() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Host:         getEnv("HTTP_HOST", "0.0.0.0"),
			Port:         getEnv("HTTP_PORT", "8082"),
			ReadTimeout:  getDuration("HTTP_READ_TIMEOUT", "5s"),
			WriteTimeout: getDuration("HTTP_WRITE_TIMEOUT", "10s"),
			IdleTimeout:  getDuration("HTTP_IDLE_TIMEOUT", "60s"),
		},
		Postgres: PostgresConfig{
			DSN:               getEnv("POSTGRES_DSN", ""),
			MaxConns:          int32(getInt("POSTGRES_MAX_CONNS", 20)),
			MinConns:          int32(getInt("POSTGRES_MIN_CONNS", 2)),
			MaxConnLifetime:   getDuration("POSTGRES_MAX_CONN_LIFETIME", "1h"),
			MaxConnIdleTime:   getDuration("POSTGRES_MAX_CONN_IDLE_TIME", "30m"),
			HealthCheckPeriod: getDuration("POSTGRES_HEALTH_CHECK_PERIOD", "1m"),
		},
		Auth: AuthConfig{
			TokenSecret: getEnv("AUTH_TOKEN_SECRET", "dev-secret"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key, fallback string) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	d, _ := time.ParseDuration(fallback)
	return d
}
