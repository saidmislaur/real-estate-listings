package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

	return nil
}

func (r *pgRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const query = `
		SELECT id, email, role
		FROM users
		WHERE email = $1
	`

	var user User

	err := r.pool.QueryRow(ctx, query, email).
		Scan(
			&user.ID,
			&user.Email,
			&user.Role,
		)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("user not found or wrong credentials",
				zap.String("email", email),
				zap.String("operation", "login"),
			)
			return nil, ErrNotFound
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			r.logger.Error("database error during login",
				zap.String("email", email),
				zap.String("pg_code", pgErr.Code),
				zap.String("pg_message", pgErr.Message),
				zap.Error(err),
			)
			return nil, fmt.Errorf("database error: %w", err)
		}

		r.logger.Error("unexpected error during login scan",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("login failed: %w", err)
	}

	return &user, nil
}

func (r *pgRepository) GetPasswordHashByEmail(ctx context.Context, email string) (string, error) {
	const query = `
		SELECT password_hash 
		FROM users 
		WHERE email = $1
	`

	var hash string
	err := r.pool.QueryRow(ctx, query, email).
		Scan(&hash)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get hash failed: %w", err)
	}
	return hash, nil
}
