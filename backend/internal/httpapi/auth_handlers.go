package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

const currentConsentVersion = "1.0"

type registerRequest struct {
	Email                string `json:"email"`
	Password             string `json:"password"`
	Language             string `json:"language"`
	AcceptedTerms        bool   `json:"acceptedTerms"`
	AcceptedPrivacy      bool   `json:"acceptedPrivacy"`
	AcceptedPersonalData bool   `json:"acceptedPersonalData"`
	ConsentVersion       string `json:"consentVersion"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User         userView `json:"user"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
}

type userView struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	HasPremium       bool       `json:"hasPremium"`
	PremiumExpiresAt *time.Time `json:"premiumExpiresAt"`
	PremiumProductID string     `json:"premiumProductId,omitempty"`
	PremiumSource    string     `json:"premiumSource,omitempty"`
	EmailVerified    bool       `json:"emailVerified"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func toUserView(u *storage.User) userView {
	return userView{
		ID:               u.ID,
		Email:            u.Email,
		HasPremium:       u.HasPremium,
		PremiumExpiresAt: u.PremiumExpiresAt,
		PremiumProductID: u.PremiumProductID,
		PremiumSource:    u.PremiumSource,
		EmailVerified:    u.EmailVerified(),
		CreatedAt:        u.CreatedAt,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(req.Email) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "invalid email format")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "password must be at least 8 characters")
		return
	}

	consentVersion := strings.TrimSpace(req.ConsentVersion)
	if consentVersion == "" {
		consentVersion = currentConsentVersion
	}
	if consentVersion != currentConsentVersion {
		writeError(w, http.StatusUnprocessableEntity, "consent_outdated", "please accept the current terms and privacy policy")
		return
	}
	if !req.AcceptedTerms || !req.AcceptedPrivacy || !req.AcceptedPersonalData {
		writeError(w, http.StatusUnprocessableEntity, "consent_required", "terms, privacy policy and personal data consent are required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to process password")
		return
	}

	ipEnc, uaEnc, err := h.sealRequestMeta(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to store access metadata")
		return
	}

	user, err := h.users.Create(r.Context(), storage.CreateUserParams{
		Email:               req.Email,
		PasswordHash:        hash,
		ConsentVersion:      consentVersion,
		ConsentIPEnc:        ipEnc,
		ConsentUserAgentEnc: uaEnc,
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "email_taken", "email is already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create user")
		return
	}

	h.recordAccess(r, user.ID, "register", ipEnc, uaEnc)

	if err = h.dispatchEmailCode(r, user.Email, storage.EmailPurposeVerify, normalizeLang(req.Language)); err != nil {
		h.writeMailError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"email":                user.Email,
		"verificationRequired": true,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		auth.CheckPasswordDummy(req.Password)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if !user.EmailVerified() {
		writeError(w, http.StatusForbidden, "email_unverified", "email is not confirmed")
		return
	}

	if ipEnc, uaEnc, err := h.sealRequestMeta(r); err != nil {
		slog.Error("encrypt login access metadata", "error", err)
	} else {
		h.recordAccess(r, user.ID, "login", ipEnc, uaEnc)
	}

	resp, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	tokenHash := auth.HashToken(req.RefreshToken)
	stored, err := h.refreshTokens.Consume(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid_token", "refresh token is invalid, expired or revoked")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to rotate token")
		return
	}

	user, err := h.users.GetByID(r.Context(), stored.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", "user not found")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "refreshToken is required")
		return
	}

	if err := h.refreshTokens.RevokeByHash(r.Context(), auth.HashToken(req.RefreshToken)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to revoke token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, toUserView(user))
}

func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	if err := h.refreshTokens.RevokeAllForUser(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to revoke sessions")
		return
	}
	if err := h.users.DeleteByID(r.Context(), userID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete account")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
