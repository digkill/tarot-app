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
	"github.com/digkill/tarot-app/backend/internal/yookassa"
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
	txns          *storage.TransactionRepo
	stats         *storage.StatsRepo
	audit         *storage.AuditRepo
	decks         *storage.DeckRepo
	usage         *storage.UsageRepo
	checkout      *storage.CheckoutRepo
	yk            *yookassa.Client
	mail          *mailer.Sender
	box           *atrest.Box
	llmClient     llm.Interpreter
	jwtSecret     jwtHolder
	limiter       *rateLimiter
	ads           *adTokenStore
}

func NewHandler(
	cfg *config.Config,
	users *storage.UserRepo,
	rt *storage.RefreshTokenRepo,
	readings *storage.ReadingRepo,
	accessStats *storage.AccessStatRepo,
	codes *storage.EmailCodeRepo,
	txns *storage.TransactionRepo,
	stats *storage.StatsRepo,
	audit *storage.AuditRepo,
	deckRepo *storage.DeckRepo,
	usageRepo *storage.UsageRepo,
	checkoutRepo *storage.CheckoutRepo,
	mail *mailer.Sender,
	box *atrest.Box,
	llmClient llm.Interpreter,
) *Handler {
	return &Handler{
		cfg:           cfg,
		users:         users,
		refreshTokens: rt,
		readings:      readings,
		accessStats:   accessStats,
		codes:         codes,
		txns:          txns,
		stats:         stats,
		audit:         audit,
		decks:         deckRepo,
		usage:         usageRepo,
		checkout:      checkoutRepo,
		yk:            yookassa.New(cfg.YooKassaShopID, cfg.YooKassaSecretKey, cfg.YooKassaVatCode),
		mail:          mail,
		box:           box,
		llmClient:     llmClient,
		jwtSecret:     jwtHolder{secret: cfg.JWTSecret},
		limiter:       newRateLimiter(),
		ads:           newAdTokenStore(),
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
