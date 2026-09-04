package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID                  string
	Email               string
	PasswordHash        string
	HasPremium          bool
	CreatedAt           time.Time
	EmailVerifiedAt     *time.Time
	TermsAcceptedAt     *time.Time
	PrivacyAcceptedAt   *time.Time
	PdnAcceptedAt       *time.Time
	ConsentVersion      string
	ConsentIPEnc        string
	ConsentUserAgentEnc string
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

const userSelectCols = `id, email, password_hash, has_premium, created_at, email_verified_at,
	terms_accepted_at, privacy_accepted_at, pdn_accepted_at, consent_version,
	consent_ip, consent_user_agent`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.HasPremium,
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
	return &u, nil
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
	return u, nil
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
	return u, nil
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
