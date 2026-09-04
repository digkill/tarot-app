package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/digkill/tarot-app/backend/internal/usage"
)

var (
	errAdTokenUnknown  = errors.New("ad token unknown")
	errAdWatchTooShort = errors.New("ad watch too short")
	errAdTokenExpired  = errors.New("ad token expired")
	errAdTokenMismatch = errors.New("ad token mismatch")
)

type adGrant struct {
	userID   string
	issuedAt time.Time
}

type adTokenStore struct {
	mu     sync.Mutex
	tokens map[string]adGrant
}

func newAdTokenStore() *adTokenStore {
	return &adTokenStore{tokens: make(map[string]adGrant)}
}

func (s *adTokenStore) issue(userID string) (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	for existing, grant := range s.tokens {
		if grant.userID == userID || time.Since(grant.issuedAt) > usage.AdTokenTTL {
			delete(s.tokens, existing)
		}
	}
	s.tokens[token] = adGrant{userID: userID, issuedAt: time.Now()}
	return token, nil
}

func (s *adTokenStore) consume(userID, token string, minWatch time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.tokens[token]
	if !ok {
		return errAdTokenUnknown
	}
	delete(s.tokens, token)
	if grant.userID != userID {
		return errAdTokenMismatch
	}
	age := time.Since(grant.issuedAt)
	if age > usage.AdTokenTTL {
		return errAdTokenExpired
	}
	if age < minWatch {
		return errAdWatchTooShort
	}
	return nil
}
