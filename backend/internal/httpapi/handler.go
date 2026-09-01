package httpapi

import (
	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/config"
	"github.com/digkill/tarot-app/backend/internal/llm"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

type jwtHolder struct {
	secret string
}

func (j jwtHolder) parse(token string) (string, error) {
	return auth.ParseAccessToken(token, j.secret)
}

type Handler struct {
	cfg           *config.Config
	users         *storage.UserRepo
	refreshTokens *storage.RefreshTokenRepo
	readings      *storage.ReadingRepo
	llmClient     *llm.Client
	jwtSecret     jwtHolder
}

func NewHandler(
	cfg *config.Config,
	users *storage.UserRepo,
	rt *storage.RefreshTokenRepo,
	readings *storage.ReadingRepo,
	llmClient *llm.Client,
) *Handler {
	return &Handler{
		cfg:           cfg,
		users:         users,
		refreshTokens: rt,
		readings:      readings,
		llmClient:     llmClient,
		jwtSecret:     jwtHolder{secret: cfg.JWTSecret},
	}
}
