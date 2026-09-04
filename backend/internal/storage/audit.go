package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID          string
	AdminUserID *string
	AdminEmail  string
	Action      string
	TargetType  string
	TargetID    string
	Details     string
	CreatedAt   time.Time
}

type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

func (r *AuditRepo) Insert(ctx context.Context, adminUserID, adminEmail, action, targetType, targetID, details string) error {
	const q = `
		INSERT INTO admin_audit (admin_user_id, admin_email, action, target_type, target_id, details)
		VALUES ($1, $2, $3, $4, $5, $6)`
	var uid any
	if adminUserID == "" {
		uid = nil
	} else {
		uid = adminUserID
	}
	_, err := r.pool.Exec(ctx, q, uid, adminEmail, action, targetType, targetID, details)
	if err != nil {
		return fmt.Errorf("insert admin audit: %w", err)
	}
	return nil
}

func (r *AuditRepo) Recent(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, admin_user_id, admin_email, action, target_type, target_id, details, created_at
		FROM admin_audit
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list admin audit: %w", err)
	}
	defer rows.Close()

	out := make([]AuditEntry, 0)
	for rows.Next() {
		var e AuditEntry
		if err = rows.Scan(&e.ID, &e.AdminUserID, &e.AdminEmail, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
