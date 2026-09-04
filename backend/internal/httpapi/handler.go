package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/digkill/tarot-app/backend/internal/atrest"
	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/config"
	"github.com/digkill/tarot-app/backend/internal/llm"
	"github.com/digkill/tarot-app/backend/internal/mailer"
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
	accessStats   *storage.AccessStatRepo
	codes         *storage.EmailCodeRepo
	mail          *mailer.Sender
	box           *atrest.Box
	llmClient     *llm.Client
	jwtSecret     jwtHolder
}

func NewHandler(
	cfg *config.Config,
	users *storage.UserRepo,
	rt *storage.RefreshTokenRepo,
	readings *storage.ReadingRepo,
	accessStats *storage.AccessStatRepo,
	codes *storage.EmailCodeRepo,
	mail *mailer.Sender,
	box *atrest.Box,
	llmClient *llm.Client,
) *Handler {
	return &Handler{
		cfg:           cfg,
		users:         users,
		refreshTokens: rt,
		readings:      readings,
		accessStats:   accessStats,
		codes:         codes,
		mail:          mail,
		box:           box,
		llmClient:     llmClient,
		jwtSecret:     jwtHolder{secret: cfg.JWTSecret},
	}
}

func (h *Handler) sealRequestMeta(r *http.Request) (ipEnc, uaEnc string, err error) {
	ipEnc, err = h.box.Seal(clientIP(r))
	if err != nil {
		return "", "", err
	}
	uaEnc, err = h.box.Seal(truncateRunes(r.Header.Get("User-Agent"), 512))
	if err != nil {
		return "", "", err
	}
	return ipEnc, uaEnc, nil
}

func (h *Handler) recordAccess(r *http.Request, userID, event, ipEnc, uaEnc string) {
	if err := h.accessStats.Insert(r.Context(), userID, event, ipEnc, uaEnc); err != nil {
		slog.Error("store access stat", "event", event, "error", err)
	}
}
