// Package auth accepts the access tokens issued by the main Moon Compass
// backend (HS256, user id in `sub`), so players sign in once for both
// services. The game server shares the JWT_SECRET and never issues tokens.
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) (*Verifier, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth: JWT secret must be at least 32 characters")
	}
	return &Verifier{secret: []byte(secret)}, nil
}

// UserID validates the signature, algorithm and expiry and returns the subject.
func (v *Verifier) UserID(token string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return v.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
