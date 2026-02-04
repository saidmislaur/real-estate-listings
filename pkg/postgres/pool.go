package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"Flatly/pkg/config"
)

func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pgConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}

	pgConfig.MaxConns = cfg.MaxConns
	pgConfig.MinConns = cfg.MinConns
	pgConfig.MaxConnLifetime = cfg.MaxConnLifetime
	pgConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	pgConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	return pgxpool.NewWithConfig(ctx, pgConfig)
}
