package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/mailer"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

const (
	emailCodeTTL      = 15 * time.Minute
	emailCodeCooldown = 60 * time.Second
)

type emailCodeRequest struct {
	Email    string `json:"email"`
	Language string `json:"language"`
}

type verifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type resetPasswordRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func normalizeLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en", "ru", "th", "zh":
		return strings.ToLower(strings.TrimSpace(lang))
	default:
		return "ru"
	}
}

func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req emailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(req.Email) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "invalid email format")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if user.EmailVerified() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if err = h.dispatchEmailCode(r, user.Email, storage.EmailPurposeVerify, normalizeLang(req.Language)); err != nil {
		h.writeMailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Code = strings.TrimSpace(req.Code)
	if !isValidEmail(req.Email) || len(req.Code) != 6 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "invalid email or code")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_code", "invalid confirmation code")
		return
	}
	if err = h.codes.Consume(r.Context(), req.Email, storage.EmailPurposeVerify, auth.HashToken(req.Code)); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_code", "invalid confirmation code")
		return
	}
	if err = h.users.MarkEmailVerified(r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to confirm email")
		return
	}
	user, err = h.users.GetByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	ipEnc, uaEnc, sealErr := h.sealRequestMeta(r)
	if sealErr == nil {
		h.recordAccess(r, user.ID, "verify", ipEnc, uaEnc)
	}

	resp, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req emailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(req.Email) {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "invalid email format")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err == nil {
		if err = h.dispatchEmailCode(r, user.Email, storage.EmailPurposeReset, normalizeLang(req.Language)); err != nil {
			if errors.Is(err, storage.ErrCodeCooldown) {
				writeJSON(w, http.StatusOK, map[string]any{"ok": true})
				return
			}
			slog.Error("send reset email", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Code = strings.TrimSpace(req.Code)
	if !isValidEmail(req.Email) || len(req.Code) != 6 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "invalid email or code")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "password must be at least 8 characters")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_code", "invalid confirmation code")
		return
	}
	if err = h.codes.Consume(r.Context(), req.Email, storage.EmailPurposeReset, auth.HashToken(req.Code)); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_code", "invalid confirmation code")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to process password")
		return
	}
	if err = h.users.UpdatePassword(r.Context(), user.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update password")
		return
	}
	_ = h.users.MarkEmailVerified(r.Context(), user.ID)
	_ = h.refreshTokens.RevokeAllForUser(r.Context(), user.ID)

	user, err = h.users.GetByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	ipEnc, uaEnc, sealErr := h.sealRequestMeta(r)
	if sealErr == nil {
		h.recordAccess(r, user.ID, "reset", ipEnc, uaEnc)
	}

	resp, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) dispatchEmailCode(r *http.Request, email, purpose, lang string) error {
	code, err := auth.GenerateOTP()
	if err != nil {
		return err
	}
	if err = h.codes.Issue(r.Context(), email, purpose, auth.HashToken(code), emailCodeTTL, emailCodeCooldown); err != nil {
		return err
	}
	kind := mailer.KindVerify
	if purpose == storage.EmailPurposeReset {
		kind = mailer.KindReset
	}
	subject, plain, htmlBody := mailer.Compose(kind, lang, code)
	if err = h.mail.Send(email, subject, plain, htmlBody); err != nil {
		_ = h.codes.DiscardLatest(r.Context(), email, purpose)
		return err
	}
	return nil
}

func (h *Handler) writeMailError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrCodeCooldown) {
		writeError(w, http.StatusTooManyRequests, "code_cooldown", "please wait before requesting another code")
		return
	}
	slog.Error("send email", "error", err)
	writeError(w, http.StatusBadGateway, "mail_failed", "failed to send email")
}
