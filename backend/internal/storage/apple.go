package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppleSubscription maps one Apple originalTransactionId to a user, and caches
// the Apple-side state we last saw. App Store Server Notifications carry no
// user id, so this mapping is how a renewal or refund finds its owner.
type AppleSubscription struct {
	OriginalTransactionID string
	UserID                string
	ProductID             string
	AppleProductID        string
	Environment           string
	Status                *int
	AutoRenewStatus       *int
	ExpiresAt             *time.Time
	LastTransactionID     string
	LastNotificationType  string
	LastNotificationUUID  string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type AppleSubRepo struct {
	pool *pgxpool.Pool
}

func NewAppleSubRepo(pool *pgxpool.Pool) *AppleSubRepo {
	return &AppleSubRepo{pool: pool}
}

const appleSubSelect = `
	original_transaction_id, user_id, product_id, apple_product_id, environment,
	status, auto_renew_status, expires_at, last_transaction_id,
	last_notification_type, last_notification_uuid, created_at, updated_at`

func scanAppleSub(row rowScanner) (*AppleSubscription, error) {
	var s AppleSubscription
	err := row.Scan(
		&s.OriginalTransactionID, &s.UserID, &s.ProductID, &s.AppleProductID, &s.Environment,
		&s.Status, &s.AutoRenewStatus, &s.ExpiresAt, &s.LastTransactionID,
		&s.LastNotificationType, &s.LastNotificationUUID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *AppleSubRepo) GetByOriginalTransactionID(ctx context.Context, id string) (*AppleSubscription, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrNotFound
	}
	const q = `SELECT ` + appleSubSelect + ` FROM apple_subscriptions WHERE original_transaction_id = $1`
	s, err := scanAppleSub(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get apple subscription: %w", err)
	}
	return s, nil
}

// Bind records the mapping and refreshes the cached Apple state. Ownership is
// first-binder-wins: user_id is deliberately never overwritten, and the row
// returned is the one now stored, so a caller detects a mismatch by comparing
// the returned UserID with its own — no separate read needed.
func (r *AppleSubRepo) Bind(ctx context.Context, s AppleSubscription) (*AppleSubscription, error) {
	if strings.TrimSpace(s.OriginalTransactionID) == "" {
		return nil, fmt.Errorf("bind apple subscription: empty original transaction id")
	}
	if strings.TrimSpace(s.Environment) == "" {
		s.Environment = "Production"
	}
	const q = `
		INSERT INTO apple_subscriptions (
			original_transaction_id, user_id, product_id, apple_product_id, environment,
			status, auto_renew_status, expires_at, last_transaction_id,
			last_notification_type, last_notification_uuid
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (original_transaction_id) DO UPDATE SET
			user_id = apple_subscriptions.user_id,
			product_id = EXCLUDED.product_id,
			apple_product_id = EXCLUDED.apple_product_id,
			environment = EXCLUDED.environment,
			status = COALESCE(EXCLUDED.status, apple_subscriptions.status),
			auto_renew_status = COALESCE(EXCLUDED.auto_renew_status, apple_subscriptions.auto_renew_status),
			expires_at = COALESCE(EXCLUDED.expires_at, apple_subscriptions.expires_at),
			last_transaction_id = COALESCE(NULLIF(EXCLUDED.last_transaction_id,''), apple_subscriptions.last_transaction_id),
			last_notification_type = COALESCE(NULLIF(EXCLUDED.last_notification_type,''), apple_subscriptions.last_notification_type),
			last_notification_uuid = COALESCE(NULLIF(EXCLUDED.last_notification_uuid,''), apple_subscriptions.last_notification_uuid),
			updated_at = NOW()
		RETURNING ` + appleSubSelect
	out, err := scanAppleSub(r.pool.QueryRow(ctx, q,
		strings.TrimSpace(s.OriginalTransactionID), s.UserID, s.ProductID, s.AppleProductID, s.Environment,
		s.Status, s.AutoRenewStatus, s.ExpiresAt, s.LastTransactionID,
		s.LastNotificationType, s.LastNotificationUUID,
	))
	if err != nil {
		return nil, fmt.Errorf("bind apple subscription: %w", err)
	}
	return out, nil
}

func (r *AppleSubRepo) ListByUser(ctx context.Context, userID string) ([]AppleSubscription, error) {
	const q = `SELECT ` + appleSubSelect + ` FROM apple_subscriptions
		WHERE user_id = $1 ORDER BY updated_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list apple subscriptions: %w", err)
	}
	defer rows.Close()

	out := make([]AppleSubscription, 0)
	for rows.Next() {
		s, err := scanAppleSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// MarkNotification records a notification uuid and reports whether it is the
// first time we have seen it. Apple retries a notification up to 5 times over
// ~3 days, so a replay must not grant or revoke twice. Same idiom as
// TransactionRepo.MarkPaidFromPending: the DB decides, not the caller.
func (r *AppleSubRepo) MarkNotification(ctx context.Context, uuid, ntype, subtype, origTxID string) (bool, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		// Nothing to de-duplicate on; treat it as fresh so the effect still applies.
		return true, nil
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO apple_notifications (notification_uuid, notification_type, subtype, original_transaction_id)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (notification_uuid) DO NOTHING`,
		uuid, ntype, subtype, strings.TrimSpace(origTxID))
	if err != nil {
		return false, fmt.Errorf("mark apple notification: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
