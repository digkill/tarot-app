package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrPremiumCodeUnavailable = errors.New("premium code invalid or already redeemed")
var ErrAlreadyLifetime = errors.New("account already has lifetime premium")

// NormalizePremiumCode accepts six ASCII letters/digits, never a client-side
// entitlement. Only the database decides whether a code can be redeemed.
func NormalizePremiumCode(raw string) (string, bool) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 6 {
		return "", false
	}
	for _, ch := range code {
		if !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') {
			return "", false
		}
	}
	return code, true
}

func PremiumCodeHash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// RedeemPremiumCode binds a code and grants lifetime premium atomically. Locks
// serialize both competing users and two different codes for the same user.
// Retrying a successful claim by the same account never grants a second time.
func (r *UserRepo) RedeemPremiumCode(ctx context.Context, userID, raw string) (*User, error) {
	code, valid := NormalizePremiumCode(raw)
	if !valid {
		return nil, ErrPremiumCodeUnavailable
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userSelectCols+` FROM users WHERE id=$1 FOR UPDATE`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var owner *string
	var redeemedAt *time.Time
	hash := PremiumCodeHash(code)
	err = tx.QueryRow(ctx, `SELECT redeemed_by, redeemed_at FROM premium_codes WHERE code_hash=$1 FOR UPDATE`, hash).Scan(&owner, &redeemedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPremiumCodeUnavailable
	}
	if err != nil {
		return nil, err
	}
	if redeemedAt != nil {
		if owner != nil && *owner == userID {
			return u, nil
		}
		return nil, ErrPremiumCodeUnavailable
	}
	if u.HasPremium && u.PremiumExpiresAt == nil {
		return nil, ErrAlreadyLifetime
	}
	if _, err = tx.Exec(ctx, `UPDATE premium_codes SET redeemed_by=$2, redeemed_at=NOW() WHERE code_hash=$1`, hash, userID); err != nil {
		return nil, err
	}
	u, err = scanUser(tx.QueryRow(ctx, `UPDATE users SET has_premium=TRUE,
		premium_product_id='premium_lifetime', premium_source='promo', premium_expires_at=NULL
		WHERE id=$1 RETURNING `+userSelectCols, userID))
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO transactions
		(user_id, product_id, kind, status, provider, provider_invoice_id, amount_kop, currency, notes)
		VALUES ($1, 'premium_lifetime', 'promo', 'paid', 'promo', $2, 0, 'RUB', 'Lifetime access code')`, userID, hash)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return u, nil
}
