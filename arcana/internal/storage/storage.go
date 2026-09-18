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
	ID          string
	Player1ID   string
	Player2ID   string
	Player1Hero string
	Player2Hero string
	Player1Deck string
	Player2Deck string
	WinnerID    string
	// The winning seat: 1, 2, or 0 for no winner. Survives a player having
	// no account row, which `WinnerID` cannot.
	WinnerSeat      int
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

// Result is win, loss or none as the given player saw it.
func (r MatchRecord) Result(userID string) string {
	seat := r.WinnerSeat
	if seat == 0 {
		switch {
		case r.WinnerID == "":
			return "none"
		case r.WinnerID == r.Player1ID:
			seat = 1
		case r.WinnerID == r.Player2ID:
			seat = 2
		default:
			return "none"
		}
	}
	switch {
	case seat == 1 && r.Player1ID == userID, seat == 2 && r.Player2ID == userID:
		return "win"
	case r.Player1ID == userID || r.Player2ID == userID:
		return "loss"
	default:
		return "none"
	}
}

type Store interface {
	SaveMatch(ctx context.Context, m MatchRecord) error
	// ListMatches is a player's finished matches, newest first, without replays.
	ListMatches(ctx context.Context, userID string, limit int) ([]MatchRecord, error)
	GetMatch(ctx context.Context, id string) (MatchRecord, error)
	// OwnsDeck reports whether the user may play with a shop deck's art: the
	// deck is free or bought. Unknown slugs are not owned.
	OwnsDeck(ctx context.Context, userID, slug string) (bool, error)
}
