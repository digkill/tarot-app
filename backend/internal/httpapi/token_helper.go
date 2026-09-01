package httpapi

import (
	"context"
	"fmt"
	"time"

	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

func (h *Handler) issueTokenPair(ctx context.Context, user *storage.User) (*authResponse, error) {
	accessToken, err := auth.GenerateAccessToken(user.ID, h.cfg.JWTSecret, h.cfg.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, hashedRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(h.cfg.RefreshTokenTTL)
	if _, err = h.refreshTokens.Create(ctx, user.ID, hashedRefresh, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &authResponse{
		User:         toUserView(user),
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
