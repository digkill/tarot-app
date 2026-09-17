package storage

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/digkill/tarot-app/arcana/internal/game"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies the Arcana migrations with a version table of their own.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return migrateDB(ctx, db)
}

func migrateDB(ctx context.Context, db *sql.DB) error {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, sub,
		goose.WithTableName("arcana_goose_db_version"))
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	_, err = provider.Up(ctx)
	return err
}

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (p *Postgres) SaveMatch(ctx context.Context, m MatchRecord) error {
	replay, err := json.Marshal(m.Replay)
	if err != nil {
		return err
	}
	// A player without an account row (a test bot) is stored as NULL, the
	// same as a deleted account, so the other player's history still saves.
	known, err := p.existingUsers(ctx, m.Player1ID, m.Player2ID)
	if err != nil {
		return err
	}
	for _, id := range []*string{&m.Player1ID, &m.Player2ID, &m.WinnerID} {
		if !known[*id] {
			*id = ""
		}
	}
	_, err = p.pool.Exec(ctx, `
		INSERT INTO arcana_matches (id, player1_id, player2_id, player1_hero, player2_hero, winner_id,
			result_reason, seed, rules_version, protocol_version, turns, started_at, finished_at,
			duration_ms, rating_delta, replay, player1_deck, player2_deck)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (id) DO NOTHING`,
		m.ID, nullable(m.Player1ID), nullable(m.Player2ID), m.Player1Hero, m.Player2Hero, nullable(m.WinnerID),
		string(m.Reason), int64(m.Seed), m.RulesVersion, m.ProtocolVersion, m.Turns, m.StartedAt, m.FinishedAt,
		m.Duration().Milliseconds(), m.RatingDelta, replay, m.Player1Deck, m.Player2Deck)
	return err
}

const matchColumns = `id, COALESCE(player1_id::text,''), COALESCE(player2_id::text,''), player1_hero, player2_hero,
	COALESCE(winner_id::text,''), result_reason, seed, rules_version, protocol_version, turns,
	started_at, finished_at, rating_delta, player1_deck, player2_deck`

func scanMatch(row pgx.Row, replay *[]byte) (MatchRecord, error) {
	var m MatchRecord
	var reason string
	var seed int64
	dest := []any{&m.ID, &m.Player1ID, &m.Player2ID, &m.Player1Hero, &m.Player2Hero, &m.WinnerID, &reason,
		&seed, &m.RulesVersion, &m.ProtocolVersion, &m.Turns, &m.StartedAt, &m.FinishedAt, &m.RatingDelta,
		&m.Player1Deck, &m.Player2Deck}
	if replay != nil {
		dest = append(dest, replay)
	}
	if err := row.Scan(dest...); err != nil {
		return m, err
	}
	m.Reason, m.Seed = game.Reason(reason), uint64(seed)
	return m, nil
}

func (p *Postgres) ListMatches(ctx context.Context, userID string, limit int) ([]MatchRecord, error) {
	rows, err := p.pool.Query(ctx, `SELECT `+matchColumns+` FROM arcana_matches
		WHERE player1_id = $1 OR player2_id = $1 ORDER BY finished_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MatchRecord
	for rows.Next() {
		m, err := scanMatch(rows, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (p *Postgres) GetMatch(ctx context.Context, id string) (MatchRecord, error) {
	var replay []byte
	m, err := scanMatch(p.pool.QueryRow(ctx, `SELECT `+matchColumns+`, replay FROM arcana_matches WHERE id = $1`, id), &replay)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(replay, &m.Replay)
	return m, err
}

func (p *Postgres) existingUsers(ctx context.Context, ids ...string) (map[string]bool, error) {
	rows, err := p.pool.Query(ctx, `SELECT id::text FROM users WHERE id::text = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	known := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		known[id] = true
	}
	return known, rows.Err()
}

func (p *Postgres) OwnsDeck(ctx context.Context, userID, slug string) (bool, error) {
	var owned bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM decks d
			WHERE d.slug = $2 AND d.is_published
			  AND (d.is_free OR EXISTS (SELECT 1 FROM user_decks ud WHERE ud.deck_id = d.id AND ud.user_id::text = $1))
		)`, userID, slug).Scan(&owned)
	return owned, err
}
