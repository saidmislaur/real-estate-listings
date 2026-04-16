package house

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Create(ctx context.Context, house *House) error {
	query := `
		INSERT INTO houses (id, address, year, developer)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, last_flat_created_at
	`

	err := r.pool.QueryRow(ctx, query, house.ID, house.Address, house.Year, house.Developer).
		Scan(&house.CreatedAt, &house.LastFlatCreatedAt)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "houses_pkey" {
		return ErrHouseExists
	}

	return fmt.Errorf("create house: %w", err)
}

func (r *PGRepository) GetByID(ctx context.Context, id int64) (*House, error) {
	query := `
		SELECT id, address, year, developer, created_at, last_flat_created_at
		FROM houses
		WHERE id = $1
	`

	var house House
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&house.ID, &house.Address, &house.Year, &house.Developer, &house.CreatedAt, &house.LastFlatCreatedAt)
	if err == nil {
		return &house, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrHouseNotFound
	}

	return nil, fmt.Errorf("get house: %w", err)
}

func (r *PGRepository) Subscribe(ctx context.Context, sub Subscription) error {
	query := `
		INSERT INTO subscriptions (house_id, user_id, email)
		VALUES ($1, NULLIF($2, 0), $3)
		ON CONFLICT (house_id, email) DO NOTHING
	`

	if _, err := r.pool.Exec(ctx, query, sub.HouseID, sub.UserID, sub.Email); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	return nil
}

func (r *PGRepository) ListSubscriberEmails(ctx context.Context, houseID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT email FROM subscriptions WHERE house_id = $1`, houseID)
	if err != nil {
		return nil, fmt.Errorf("query subscribers: %w", err)
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}
