package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User         userView `json:"user"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
}

type userView struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	HasPremium bool      `json:"hasPremium"`
	CreatedAt  time.Time `json:"createdAt"`
}

func toUserView(u *storage.User) userView {
	return userView{
		ID:         u.ID,
		Email:      u.Email,
		HasPremium: u.HasPremium,
		CreatedAt:  u.CreatedAt,
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

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to process password")
		return
	}

	user, err := h.users.Create(r.Context(), req.Email, hash)
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "email_taken", "email is already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create user")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
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

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, toUserView(user))
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
