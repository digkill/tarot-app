package storage

import (
	"context"
	"sort"
	"sync"
)

// Memory is an in-process Store for tests and local runs without a database.
type Memory struct {
	mu      sync.Mutex
	matches map[string]MatchRecord
}

func NewMemory() *Memory { return &Memory{matches: map[string]MatchRecord{}} }

func (s *Memory) SaveMatch(_ context.Context, m MatchRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.matches[m.ID]; !ok {
		s.matches[m.ID] = m
	}
	return nil
}

func (s *Memory) ListMatches(_ context.Context, userID string, limit int) ([]MatchRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []MatchRecord
	for _, m := range s.matches {
		if m.Player1ID == userID || m.Player2ID == userID {
			m.Replay = nil
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FinishedAt.After(out[j].FinishedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// OwnsDeck accepts everything: there is no shop without a database.
func (s *Memory) OwnsDeck(context.Context, string, string) (bool, error) { return true, nil }

func (s *Memory) GetMatch(_ context.Context, id string) (MatchRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.matches[id]
	if !ok {
		return m, ErrNotFound
	}
	return m, nil
}
