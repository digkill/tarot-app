package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EmailPurposeVerify = "verify"
	EmailPurposeReset  = "reset"
)

var ErrCodeCooldown = errors.New("code cooldown")
var ErrInvalidCode = errors.New("invalid code")

type EmailCodeRepo struct {
	pool *pgxpool.Pool
}

func NewEmailCodeRepo(pool *pgxpool.Pool) *EmailCodeRepo {
	return &EmailCodeRepo{pool: pool}
}

func (r *EmailCodeRepo) Issue(ctx context.Context, email, purpose, codeHash string, ttl, cooldown time.Duration) error {
	var lastCreated time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT created_at FROM email_codes
		WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1`, email, purpose).Scan(&lastCreated)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check email code cooldown: %w", err)
	}
	if err == nil && time.Since(lastCreated) < cooldown {
		return ErrCodeCooldown
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email code tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		UPDATE email_codes
		SET consumed_at = NOW()
		WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL`, email, purpose); err != nil {
		return fmt.Errorf("invalidate previous email codes: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO email_codes (email, purpose, code_hash, expires_at)
		VALUES ($1, $2, $3, $4)`, email, purpose, codeHash, time.Now().Add(ttl)); err != nil {
		return fmt.Errorf("insert email code: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit email code: %w", err)
	}
	return nil
}

func (r *EmailCodeRepo) Consume(ctx context.Context, email, purpose, codeHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin consume email code: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id, storedHash string
	var attempts int
	err = tx.QueryRow(ctx, `
		SELECT id, code_hash, attempts FROM email_codes
		WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE`, email, purpose).Scan(&id, &storedHash, &attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidCode
		}
		return fmt.Errorf("lock email code: %w", err)
	}

	attempts++
	valid := storedHash == codeHash && attempts <= 5
	consumed := valid || attempts >= 5
	if _, err = tx.Exec(ctx, `
		UPDATE email_codes
		SET attempts = $2, consumed_at = CASE WHEN $3 THEN NOW() ELSE consumed_at END
		WHERE id = $1`, id, attempts, consumed); err != nil {
		return fmt.Errorf("update email code: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit consume email code: %w", err)
	}
	if !valid {
		return ErrInvalidCode
	}
	return nil
}

func (r *EmailCodeRepo) DiscardLatest(ctx context.Context, email, purpose string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE email_codes
		SET consumed_at = NOW()
		WHERE id = (
			SELECT id FROM email_codes
			WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		)`, email, purpose)
	if err != nil {
		return fmt.Errorf("discard email code: %w", err)
	}
	return nil
}
