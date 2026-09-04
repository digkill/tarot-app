package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrQuotaExceeded  = errors.New("quota exceeded")
	ErrActionCooldown = errors.New("action cooldown")
)

type DailyUsage struct {
	DailyCards      int
	Interpretations int
	AdUnlocks       int
	LastAdAt        *time.Time
	LastInterpretAt *time.Time
}

type UsageRepo struct {
	pool *pgxpool.Pool
}

func NewUsageRepo(pool *pgxpool.Pool) *UsageRepo {
	return &UsageRepo{pool: pool}
}

func (r *UsageRepo) Get(ctx context.Context, userID string, day time.Time) (DailyUsage, error) {
	var row DailyUsage
	err := r.pool.QueryRow(ctx, `
		SELECT daily_cards, interpretations, ad_unlocks, last_ad_at, last_interpret_at
		FROM daily_usage
		WHERE user_id = $1 AND day = $2::date`, userID, day).Scan(
		&row.DailyCards, &row.Interpretations, &row.AdUnlocks, &row.LastAdAt, &row.LastInterpretAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyUsage{}, nil
	}
	if err != nil {
		return DailyUsage{}, fmt.Errorf("get daily usage: %w", err)
	}
	return row, nil
}

func (r *UsageRepo) IncrementDailyCards(ctx context.Context, userID string, day time.Time, limit int) (DailyUsage, error) {
	row, err := r.increment(ctx, userID, day, `
		INSERT INTO daily_usage (user_id, day, daily_cards)
		VALUES ($1, $2::date, 1)
		ON CONFLICT (user_id, day) DO UPDATE
		SET daily_cards = daily_usage.daily_cards + 1
		WHERE daily_usage.daily_cards < $3
		RETURNING daily_cards, interpretations, ad_unlocks, last_ad_at, last_interpret_at`, limit)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DailyUsage{}, ErrQuotaExceeded
		}
		return DailyUsage{}, err
	}
	return row, nil
}

func (r *UsageRepo) IncrementAdCard(ctx context.Context, userID string, day time.Time, cardLimit, adLimit int, cooldownUntil time.Time) (DailyUsage, error) {
	row, err := r.increment(ctx, userID, day, `
		INSERT INTO daily_usage (user_id, day, daily_cards, ad_unlocks, last_ad_at)
		VALUES ($1, $2::date, 1, 1, NOW())
		ON CONFLICT (user_id, day) DO UPDATE
		SET daily_cards = daily_usage.daily_cards + 1,
		    ad_unlocks = daily_usage.ad_unlocks + 1,
		    last_ad_at = NOW()
		WHERE daily_usage.daily_cards < $3
		  AND daily_usage.ad_unlocks < $4
		  AND (daily_usage.last_ad_at IS NULL OR daily_usage.last_ad_at <= $5)
		RETURNING daily_cards, interpretations, ad_unlocks, last_ad_at, last_interpret_at`,
		cardLimit, adLimit, cooldownUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			current, getErr := r.Get(ctx, userID, day)
			if getErr != nil {
				return DailyUsage{}, getErr
			}
			if current.LastAdAt != nil && current.LastAdAt.After(cooldownUntil) {
				return current, ErrActionCooldown
			}
			return current, ErrQuotaExceeded
		}
		return DailyUsage{}, err
	}
	return row, nil
}

func (r *UsageRepo) IncrementInterpretation(ctx context.Context, userID string, day time.Time, limit int, cooldownUntil time.Time) (DailyUsage, error) {
	if limit <= 0 {
		return DailyUsage{}, ErrQuotaExceeded
	}
	row, err := r.increment(ctx, userID, day, `
		INSERT INTO daily_usage (user_id, day, interpretations, last_interpret_at)
		VALUES ($1, $2::date, 1, NOW())
		ON CONFLICT (user_id, day) DO UPDATE
		SET interpretations = daily_usage.interpretations + 1,
		    last_interpret_at = NOW()
		WHERE daily_usage.interpretations < $3
		  AND (daily_usage.last_interpret_at IS NULL OR daily_usage.last_interpret_at <= $4)
		RETURNING daily_cards, interpretations, ad_unlocks, last_ad_at, last_interpret_at`,
		limit, cooldownUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			current, getErr := r.Get(ctx, userID, day)
			if getErr != nil {
				return DailyUsage{}, getErr
			}
			if current.LastInterpretAt != nil && current.LastInterpretAt.After(cooldownUntil) {
				return current, ErrActionCooldown
			}
			return current, ErrQuotaExceeded
		}
		return DailyUsage{}, err
	}
	return row, nil
}

func (r *UsageRepo) increment(ctx context.Context, userID string, day time.Time, sql string, args ...any) (DailyUsage, error) {
	all := append([]any{userID, day}, args...)
	var row DailyUsage
	err := r.pool.QueryRow(ctx, sql, all...).Scan(
		&row.DailyCards, &row.Interpretations, &row.AdUnlocks, &row.LastAdAt, &row.LastInterpretAt,
	)
	if err != nil {
		return DailyUsage{}, err
	}
	return row, nil
}
