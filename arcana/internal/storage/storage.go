// Package storage keeps finished matches. Live matches never touch the
// database: they are held in memory and written once, when they end.
package storage

import (
	"context"
	"errors"
	"time"

	"github.com/digkill/tarot-app/arcana/internal/game"
)

var ErrNotFound = errors.New("not found")

type MatchRecord struct {
	ID              string
	Player1ID       string
	Player2ID       string
	Player1Hero     string
	Player2Hero     string
	WinnerID        string
	Reason          game.Reason
	Seed            uint64
	RulesVersion    int
	ProtocolVersion int
	Turns           int
	StartedAt       time.Time
	FinishedAt      time.Time
	RatingDelta     *int
	Replay          []game.Entry
}

func (r MatchRecord) Duration() time.Duration { return r.FinishedAt.Sub(r.StartedAt) }

type Store interface {
	SaveMatch(ctx context.Context, m MatchRecord) error
	// ListMatches is a player's finished matches, newest first, without replays.
	ListMatches(ctx context.Context, userID string, limit int) ([]MatchRecord, error)
	GetMatch(ctx context.Context, id string) (MatchRecord, error)
}
