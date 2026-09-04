package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessStatRepo struct {
	pool *pgxpool.Pool
}

func NewAccessStatRepo(pool *pgxpool.Pool) *AccessStatRepo {
	return &AccessStatRepo{pool: pool}
}

func (r *AccessStatRepo) Insert(ctx context.Context, userID, event, ipEnc, userAgentEnc string) error {
	const q = `
		INSERT INTO access_stats (user_id, event, ip_enc, user_agent_enc)
		VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, q, userID, event, ipEnc, userAgentEnc)
	if err != nil {
		return fmt.Errorf("insert access stat: %w", err)
	}
	return nil
}
