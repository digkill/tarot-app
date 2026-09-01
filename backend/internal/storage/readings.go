package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReadingItem struct {
	PositionIndex int    `json:"positionIndex"`
	CardID        string `json:"cardId"`
	IsReversed    bool   `json:"isReversed"`
}

type Reading struct {
	ID          string
	UserID      string
	SpreadID    string
	DeckID      string
	Items       []ReadingItem
	SummaryText string
	Notes       string
	Favorite    bool
	CreatedAt   time.Time
}

type ReadingRepo struct {
	pool *pgxpool.Pool
}

func NewReadingRepo(pool *pgxpool.Pool) *ReadingRepo {
	return &ReadingRepo{pool: pool}
}

type CreateReadingParams struct {
	UserID      string
	SpreadID    string
	DeckID      string
	Items       []ReadingItem
	SummaryText string
	Notes       string
}

func (r *ReadingRepo) Create(ctx context.Context, p CreateReadingParams) (*Reading, error) {
	itemsJSON, err := json.Marshal(p.Items)
	if err != nil {
		return nil, fmt.Errorf("marshal reading items: %w", err)
	}

	const q = `
		INSERT INTO readings (user_id, spread_id, deck_id, items, summary_text, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, spread_id, deck_id, items, summary_text, notes, favorite, created_at`

	return r.scanOne(r.pool.QueryRow(ctx, q,
		p.UserID, p.SpreadID, p.DeckID, itemsJSON, p.SummaryText, p.Notes,
	))
}

func (r *ReadingRepo) List(ctx context.Context, userID string, limit, offset int) ([]Reading, error) {
	const q = `
		SELECT id, user_id, spread_id, deck_id, items, summary_text, notes, favorite, created_at
		FROM readings
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list readings: %w", err)
	}
	defer rows.Close()

	readings := make([]Reading, 0)
	for rows.Next() {
		rd, err := r.scanOne(rows)
		if err != nil {
			return nil, err
		}
		readings = append(readings, *rd)
	}
	return readings, rows.Err()
}

func (r *ReadingRepo) GetByID(ctx context.Context, id, userID string) (*Reading, error) {
	const q = `
		SELECT id, user_id, spread_id, deck_id, items, summary_text, notes, favorite, created_at
		FROM readings
		WHERE id = $1 AND user_id = $2`

	rd, err := r.scanOne(r.pool.QueryRow(ctx, q, id, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rd, nil
}

type UpdateReadingParams struct {
	Notes    *string
	Favorite *bool
}

func (r *ReadingRepo) Update(ctx context.Context, id, userID string, p UpdateReadingParams) (*Reading, error) {
	const q = `
		UPDATE readings
		SET
			notes    = COALESCE($3, notes),
			favorite = COALESCE($4, favorite)
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, spread_id, deck_id, items, summary_text, notes, favorite, created_at`

	rd, err := r.scanOne(r.pool.QueryRow(ctx, q, id, userID, p.Notes, p.Favorite))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rd, nil
}

func (r *ReadingRepo) Delete(ctx context.Context, id, userID string) error {
	const q = `DELETE FROM readings WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("delete reading: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *ReadingRepo) scanOne(row rowScanner) (*Reading, error) {
	var rd Reading
	var itemsRaw []byte
	err := row.Scan(
		&rd.ID, &rd.UserID, &rd.SpreadID, &rd.DeckID,
		&itemsRaw, &rd.SummaryText, &rd.Notes, &rd.Favorite, &rd.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(itemsRaw, &rd.Items); err != nil {
		return nil, fmt.Errorf("unmarshal reading items: %w", err)
	}
	return &rd, nil
}
