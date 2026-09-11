package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`

	var t RefreshToken
	err := r.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert refresh token: %w", err)
	}
	return &t, nil
}

// consumeGracePeriod lets a refresh token be reused for a short window after
// its first rotation. Mobile clients sometimes never see the response to a
// successful /auth/refresh call (backgrounded app, flaky network) and retry
// with the same now-rotated token; without this, that retry is rejected as
// invalid and the user is forced to log in again even though the refresh
// actually succeeded. The token is still single-use beyond the grace window.
const consumeGracePeriod = 60 * time.Second

// Consume revokes a live token on first use and returns it; a retry with the
// same token within consumeGracePeriod of that first revocation succeeds
// again instead of failing (see consumeGracePeriod). Concurrent or replayed
// use outside that window is rejected.
func (r *RefreshTokenRepo) Consume(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE token_hash = $1
		  AND expires_at > NOW()
		  AND (revoked_at IS NULL OR revoked_at > $2)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`

	graceCutoff := time.Now().Add(-consumeGracePeriod)
	var t RefreshToken
	err := r.pool.QueryRow(ctx, q, tokenHash, graceCutoff).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consume refresh token: %w", err)
	}
	return &t, nil
}

func (r *RefreshTokenRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}
