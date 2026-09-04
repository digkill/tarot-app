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

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID                  string
	Email               string
	PasswordHash        string
	HasPremium          bool
	PremiumExpiresAt    *time.Time
	PremiumProductID    string
	PremiumSource       string
	Role                string
	CreatedAt           time.Time
	EmailVerifiedAt     *time.Time
	TermsAcceptedAt     *time.Time
	PrivacyAcceptedAt   *time.Time
	PdnAcceptedAt       *time.Time
	ConsentVersion      string
	ConsentIPEnc        string
	ConsentUserAgentEnc string
}

type AdminUser struct {
	ID              string
	Email           string
	HasPremium      bool
	Role            string
	CreatedAt       time.Time
	EmailVerifiedAt *time.Time
}

type CreateUserParams struct {
	Email               string
	PasswordHash        string
	ConsentVersion      string
	ConsentIPEnc        string
	ConsentUserAgentEnc string
}

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

const userSelectCols = `id, email, password_hash, has_premium, premium_expires_at, premium_product_id, premium_source,
	role, created_at, email_verified_at,
	terms_accepted_at, privacy_accepted_at, pdn_accepted_at, consent_version,
	consent_ip, consent_user_agent`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	var productID, source *string
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.HasPremium,
		&u.PremiumExpiresAt,
		&productID,
		&source,
		&u.Role,
		&u.CreatedAt,
		&u.EmailVerifiedAt,
		&u.TermsAcceptedAt,
		&u.PrivacyAcceptedAt,
		&u.PdnAcceptedAt,
		&u.ConsentVersion,
		&u.ConsentIPEnc,
		&u.ConsentUserAgentEnc,
	)
	if err != nil {
		return nil, err
	}
	if productID != nil {
		u.PremiumProductID = *productID
	}
	if source != nil {
		u.PremiumSource = *source
	}
	if u.Role == "" {
		u.Role = RoleUser
	}
	return &u, nil
}

func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

func (r *UserRepo) Create(ctx context.Context, p CreateUserParams) (*User, error) {
	const q = `
		INSERT INTO users (
			email, password_hash, terms_accepted_at, privacy_accepted_at, pdn_accepted_at,
			consent_version, consent_ip, consent_user_agent
		)
		VALUES ($1, $2, NOW(), NOW(), NOW(), $3, $4, $5)
		RETURNING ` + userSelectCols

	u, err := scanUser(r.pool.QueryRow(ctx, q, p.Email, p.PasswordHash, p.ConsentVersion, p.ConsentIPEnc, p.ConsentUserAgentEnc))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	const q = `SELECT ` + userSelectCols + ` FROM users WHERE email = $1`
	u, err := scanUser(r.pool.QueryRow(ctx, q, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return r.applyExpiry(ctx, u)
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	const q = `SELECT ` + userSelectCols + ` FROM users WHERE id = $1`
	u, err := scanUser(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return r.applyExpiry(ctx, u)
}

func (r *UserRepo) DeleteByID(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (u *User) EmailVerified() bool {
	return u != nil && u.EmailVerifiedAt != nil
}

func (r *UserRepo) MarkEmailVerified(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET email_verified_at = COALESCE(email_verified_at, NOW())
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) SetPremium(ctx context.Context, id string, premium bool) error {
	if premium {
		return r.GrantPremium(ctx, id, PremiumGrant{Source: "admin"})
	}
	return r.RevokePremium(ctx, id)
}

type PremiumGrant struct {
	ProductID string
	Source    string
	ExpiresAt *time.Time
}

func (r *UserRepo) GrantPremium(ctx context.Context, id string, g PremiumGrant) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET has_premium = TRUE,
		    premium_product_id = NULLIF($2, ''),
		    premium_source = NULLIF($3, ''),
		    premium_expires_at = $4
		WHERE id = $1`, id, g.ProductID, g.Source, g.ExpiresAt)
	if err != nil {
		return fmt.Errorf("grant premium: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) RevokePremium(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET has_premium = FALSE,
		    premium_product_id = NULL,
		    premium_source = NULL,
		    premium_expires_at = NULL
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("revoke premium: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) ExpireDue(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET has_premium = FALSE,
		    premium_product_id = NULL,
		    premium_source = NULL,
		    premium_expires_at = NULL
		WHERE has_premium = TRUE
		  AND premium_expires_at IS NOT NULL
		  AND premium_expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("expire premium: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *UserRepo) applyExpiry(ctx context.Context, u *User) (*User, error) {
	if u == nil || !u.HasPremium || u.PremiumExpiresAt == nil || !time.Now().After(*u.PremiumExpiresAt) {
		return u, nil
	}
	if err := r.RevokePremium(ctx, u.ID); err != nil {
		return u, err
	}
	u.HasPremium = false
	u.PremiumExpiresAt = nil
	u.PremiumProductID = ""
	u.PremiumSource = ""
	return u, nil
}

func (r *UserRepo) SetRole(ctx context.Context, id, role string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE id = $1`, id, role)
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) ListForAdmin(ctx context.Context, query string, limit, offset int) ([]AdminUser, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	query = strings.TrimSpace(query)

	const countQ = `
		SELECT COUNT(*) FROM users
		WHERE $1 = '' OR email ILIKE '%' || $1 || '%'`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	const q = `
		SELECT id, email, has_premium, role, created_at, email_verified_at
		FROM users
		WHERE $1 = '' OR email ILIKE '%' || $1 || '%'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	out := make([]AdminUser, 0)
	for rows.Next() {
		var u AdminUser
		if err = rows.Scan(&u.ID, &u.Email, &u.HasPremium, &u.Role, &u.CreatedAt, &u.EmailVerifiedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

type EnsureAdminParams struct {
	Email        string
	PasswordHash string
}

func (r *UserRepo) EnsureAdmin(ctx context.Context, p EnsureAdminParams) (created bool, err error) {
	existing, err := r.GetByEmail(ctx, p.Email)
	if err == nil {
		if existing.Role != RoleAdmin {
			if err = r.SetRole(ctx, existing.ID, RoleAdmin); err != nil {
				return false, err
			}
		}
		return false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return false, err
	}

	const q = `
		INSERT INTO users (
			email, password_hash, has_premium, premium_source, role, email_verified_at,
			terms_accepted_at, privacy_accepted_at, pdn_accepted_at, consent_version
		)
		VALUES ($1, $2, TRUE, 'admin', $3, NOW(), NOW(), NOW(), NOW(), '1.0')
		RETURNING ` + userSelectCols
	_, err = scanUser(r.pool.QueryRow(ctx, q, p.Email, p.PasswordHash, RoleAdmin))
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, fmt.Errorf("insert admin: %w", err)
	}
	return true, nil
}
