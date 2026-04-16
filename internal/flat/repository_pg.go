package flat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"Flatly/internal/house"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Create(ctx context.Context, flat *Flat) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO flats (house_id, number, price, rooms, status, created_by)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, 0))
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query, flat.HouseID, flat.Number, flat.Price, flat.Rooms, flat.Status, flat.CreatedBy).
		Scan(&flat.ID, &flat.CreatedAt, &flat.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "flats_house_id_number_key":
				return ErrFlatAlreadyExists
			case "flats_house_id_fkey":
				return house.ErrHouseNotFound
			}
		}
		return fmt.Errorf("create flat: %w", err)
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE houses SET last_flat_created_at = $2 WHERE id = $1`, flat.HouseID, now); err != nil {
		return fmt.Errorf("touch house: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	flat.UpdatedAt = flat.CreatedAt
	return nil
}

func (r *PGRepository) UpdateStatus(ctx context.Context, flatID int64, status Status, moderatorID int64) (*Flat, error) {
	var (
		query string
		args  []any
	)

	switch status {
	case StatusOnModeration:
		query = `
			UPDATE flats
			SET status = $2, moderation_by = NULLIF($3, 0), updated_at = now()
			WHERE id = $1 AND status = 'created' AND moderation_by IS NULL
			RETURNING id, house_id, number, price, rooms, status, created_by, moderation_by, created_at, updated_at
		`
		args = []any{flatID, status, moderatorID}
	case StatusApproved, StatusDeclined:
		query = `
			UPDATE flats
			SET status = $2, updated_at = now()
			WHERE id = $1 AND status = 'on moderation' AND moderation_by = NULLIF($3, 0)
			RETURNING id, house_id, number, price, rooms, status, created_by, moderation_by, created_at, updated_at
		`
		args = []any{flatID, status, moderatorID}
	default:
		return nil, ErrInvalidStatus
	}

	var flat Flat
	err := r.pool.QueryRow(ctx, query, args...).
		Scan(&flat.ID, &flat.HouseID, &flat.Number, &flat.Price, &flat.Rooms, &flat.Status, &flat.CreatedBy, &flat.ModeratorID, &flat.CreatedAt, &flat.UpdatedAt)
	if err == nil {
		return &flat, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if exists, existsErr := r.flatExists(ctx, flatID); existsErr == nil && exists {
			return nil, ErrModerationConflict
		}
		return nil, ErrFlatNotFound
	}

	return nil, fmt.Errorf("update flat status: %w", err)
}

func (r *PGRepository) ListByHouse(ctx context.Context, houseID int64, includeAll bool) ([]house.FlatSummary, error) {
	query := `
		SELECT id, house_id, number, price, rooms, status, created_at, COALESCE(created_by, 0), COALESCE(moderation_by, 0)
		FROM flats
		WHERE house_id = $1
	`
	args := []any{houseID}
	if !includeAll {
		query += ` AND status = 'approved'`
	}
	query += ` ORDER BY created_at DESC, id DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list flats by house: %w", err)
	}
	defer rows.Close()

	flats := make([]house.FlatSummary, 0)
	for rows.Next() {
		var item house.FlatSummary
		if err := rows.Scan(&item.ID, &item.HouseID, &item.Number, &item.Price, &item.Rooms, &item.Status, &item.CreatedAt, &item.CreatedBy, &item.ModeratorID); err != nil {
			return nil, fmt.Errorf("scan flat summary: %w", err)
		}
		flats = append(flats, item)
	}

	return flats, rows.Err()
}

func (r *PGRepository) flatExists(ctx context.Context, flatID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM flats WHERE id = $1)`, flatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check flat exists: %w", err)
	}
	return exists, nil
}
