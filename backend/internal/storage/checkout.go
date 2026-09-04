package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckoutSession struct {
	TransactionID     string
	ConfirmationURL   string
	ProviderPaymentID string
}

type CheckoutRepo struct {
	pool *pgxpool.Pool
}

func NewCheckoutRepo(pool *pgxpool.Pool) *CheckoutRepo {
	return &CheckoutRepo{pool: pool}
}

func (r *CheckoutRepo) Upsert(ctx context.Context, s CheckoutSession) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO checkout_sessions (transaction_id, confirmation_url, provider_payment_id)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (transaction_id) DO UPDATE
		SET confirmation_url = EXCLUDED.confirmation_url,
		    provider_payment_id = COALESCE(EXCLUDED.provider_payment_id, checkout_sessions.provider_payment_id)`,
		s.TransactionID, s.ConfirmationURL, s.ProviderPaymentID)
	if err != nil {
		return fmt.Errorf("upsert checkout session: %w", err)
	}
	return nil
}

func (r *CheckoutRepo) GetByTransaction(ctx context.Context, txID string) (*CheckoutSession, error) {
	var s CheckoutSession
	var payID *string
	err := r.pool.QueryRow(ctx, `
		SELECT transaction_id, confirmation_url, provider_payment_id
		FROM checkout_sessions WHERE transaction_id = $1`, txID).Scan(&s.TransactionID, &s.ConfirmationURL, &payID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get checkout session: %w", err)
	}
	if payID != nil {
		s.ProviderPaymentID = *payID
	}
	return &s, nil
}
