package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	// "github.com/jackc/pgx/v5/pgconn"
)

type pgRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPgRepository(pool *pgxpool.Pool, logger *zap.Logger) Repository {
	l := logger.Named("user.pgRepository").With(zap.String("component", "user.pgRepository"))
	return &pgRepository{
		pool:   pool,
		logger: l,
	}
}

func (r *pgRepository) Create(ctx context.Context, u *User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		u.Email,
		u.passwordHash,
		u.Role,
	)

	if err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "users_email_key" {
				r.logger.Info("user create failed: email exists",
					zap.String("email", u.Email),
					zap.String("pg_constraint", pgErr.ConstraintName),
					zap.String("pg_code", pgErr.Code),
				)
				return ErrEmailExists
			}

			r.logger.Error("user create failed: postgres error",
				zap.String("email", u.Email),
				zap.String("pg_constraint", pgErr.ConstraintName),
				zap.String("pg_code", pgErr.Code),
				zap.String("pg_message", pgErr.Message),
				zap.Error(err),
			)
			return fmt.Errorf("postgres error: %w", err)
		}

		r.logger.Error("не удалось создать пользователя",
			zap.String("login", u.Email),
			zap.String("operation", "create_user"),
			zap.Error(err),
		)
		return fmt.Errorf("create user failed: %w", err)

	}

	r.logger.Info("пользователь успешно создан",
		zap.Int("user id", u.ID),
		zap.String("email", u.Email),
		zap.String("роль", string(u.Role)),
	)

	return nil
}
