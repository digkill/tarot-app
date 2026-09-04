package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/digkill/tarot-app/backend/internal/decks"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

func (h *Handler) AdminDecks(w http.ResponseWriter, r *http.Request) {
	list, err := h.decks.ListAll(r.Context())
	if err != nil {
		http.Error(w, "failed to list decks", http.StatusInternalServerError)
		return
	}
	base := h.adminBase(r, "Колоды", "decks")
	h.renderAdmin(w, "decks", http.StatusOK, struct {
		adminBase
		Decks []storage.Deck
	}{base, list})
}

func presetNames() []string {
	return []string{"classic", "japanese", "ink", "forest"}
}

func (h *Handler) AdminDeckNew(w http.ResponseWriter, r *http.Request) {
	base := h.adminBase(r, "Новая колода", "decks")
	h.renderAdmin(w, "deck_edit", http.StatusOK, struct {
		adminBase
		Deck    *storage.Deck
		Presets []string
		Cover   string
		Back    string
		Cards   map[string]string
		IsNew   bool
	}{base, nil, presetNames(), "", "", map[string]string{}, true})
}

func (h *Handler) AdminDeck(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.decks.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to load deck", http.StatusInternalServerError)
		return
	}
	cover, back, cards := h.deckCardURLs(d.Slug)
	base := h.adminBase(r, d.Slug, "decks")
	h.renderAdmin(w, "deck_edit", http.StatusOK, struct {
		adminBase
		Deck    *storage.Deck
		Presets []string
		Cover   string
		Back    string
		Cards   map[string]string
		IsNew   bool
	}{base, d, presetNames(), cover, back, cards, false})
}

func formI18n(r *http.Request, prefix string) storage.I18nMap {
	out := storage.I18nMap{}
	for _, lang := range []string{"ru", "en", "th", "zh"} {
		if v := strings.TrimSpace(r.FormValue(prefix + "_" + lang)); v != "" {
			out[lang] = v
		}
	}
	return out
}

func formTheme(r *http.Request) decks.Theme {
	if preset := strings.TrimSpace(r.FormValue("theme_preset")); preset != "" {
		if t, ok := decks.Presets[preset]; ok && r.FormValue("use_preset") == "1" {
			return t
		}
	}
	return decks.NormalizeTheme(decks.Theme{
		Bg:     r.FormValue("theme_bg"),
		Panel:  r.FormValue("theme_panel"),
		Accent: r.FormValue("theme_accent"),
		Text:   r.FormValue("theme_text"),
		Muted:  r.FormValue("theme_muted"),
		Gold:   r.FormValue("theme_gold"),
		Danger: r.FormValue("theme_danger"),
		TabBar: r.FormValue("theme_tabbar"),
	})
}

func parsePriceKop(r *http.Request) int {
	raw := strings.ReplaceAll(strings.TrimSpace(r.FormValue("price_rub")), ",", ".")
	if raw == "" {
		return 0
	}
	rub, err := strconv.ParseFloat(raw, 64)
	if err != nil || rub < 0 {
		return 0
	}
	return int(rub * 100)
}

func deckParamsFromForm(r *http.Request, slug string) storage.UpsertDeckParams {
	product := storage.NullIfBlank(r.FormValue("rustore_product_id"))
	if product == nil {
		product = storage.NullIfBlank(decks.ProductIDForSlug(slug))
	}
	return storage.UpsertDeckParams{
		Slug:             slug,
		TitleI18n:        formI18n(r, "title"),
		DescriptionI18n:  formI18n(r, "description"),
		Theme:            formTheme(r),
		PriceKop:         parsePriceKop(r),
		RustoreProductID: product,
		IsFree:           r.FormValue("is_free") == "1",
		IsPublished:      r.FormValue("is_published") == "1",
		SortOrder:        atoiDefault(r.FormValue("sort_order"), 0),
	}
}

func atoiDefault(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}

func (h *Handler) AdminDeckCreate(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	if !slugRe.MatchString(slug) {
		http.Redirect(w, r, "/admin/decks/new?err=validation", http.StatusFound)
		return
	}
	d, err := h.decks.Create(r.Context(), deckParamsFromForm(r, slug))
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			http.Redirect(w, r, "/admin/decks/new?err=conflict", http.StatusFound)
			return
		}
		slog.Error("create deck", "error", err)
		http.Error(w, "failed to create deck", http.StatusInternalServerError)
		return
	}
	_ = os.MkdirAll(h.deckDir(d.Slug), 0o755)
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "deck_create", "deck", d.ID, d.Slug)
	http.Redirect(w, r, "/admin/decks/"+d.ID+"?ok=deck_saved", http.StatusFound)
}

func (h *Handler) AdminDeckUpdate(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	existing, err := h.decks.GetByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/decks?err=not_found", http.StatusFound)
		return
	}
	p := deckParamsFromForm(r, existing.Slug)
	if _, err = h.decks.Update(r.Context(), id, p); err != nil {
		slog.Error("update deck", "error", err)
		http.Error(w, "failed to update deck", http.StatusInternalServerError)
		return
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "deck_update", "deck", id, existing.Slug)
	http.Redirect(w, r, "/admin/decks/"+id+"?ok=deck_saved", http.StatusFound)
}

func (h *Handler) AdminDeckImport(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	d, err := h.decks.GetByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/decks?err=not_found", http.StatusFound)
		return
	}
	if err = r.ParseMultipartForm(512 << 20); err != nil {
		http.Redirect(w, r, "/admin/decks/"+id+"?err=validation", http.StatusFound)
		return
	}
	file, header, err := r.FormFile("archive")
	if err != nil {
		http.Redirect(w, r, "/admin/decks/"+id+"?err=validation", http.StatusFound)
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "tarot-deck-*.zip")
	if err != nil {
		http.Error(w, "temp file", http.StatusInternalServerError)
		return
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err = io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}
	_ = tmp.Close()

	dest := h.deckDir(d.Slug)
	res, err := decks.ImportZip(tmpName, dest)
	if err != nil {
		slog.Error("import deck zip", "name", header.Filename, "error", err)
		http.Redirect(w, r, "/admin/decks/"+id+"?err=import", http.StatusFound)
		return
	}
	if err = h.decks.SetImportMeta(r.Context(), d.ID, res.Cards, res.Back); err != nil {
		slog.Error("set import meta", "error", err)
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "deck_import", "deck", d.ID,
		d.Slug+" cards="+strconv.Itoa(res.Cards))
	http.Redirect(w, r, "/admin/decks/"+id+"?ok=imported", http.StatusFound)
}

func (h *Handler) AdminDeckGrant(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	d, err := h.decks.GetByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/decks?err=not_found", http.StatusFound)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		http.Redirect(w, r, "/admin/decks/"+id+"?err=not_found", http.StatusFound)
		return
	}
	if err = h.decks.Grant(r.Context(), user.ID, d.ID, "admin", nil); err != nil {
		http.Error(w, "failed to grant", http.StatusInternalServerError)
		return
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "deck_grant", "deck", d.ID, user.Email)
	http.Redirect(w, r, "/admin/decks/"+id+"?ok=deck_granted", http.StatusFound)
}

func (h *Handler) AdminDeckGrantOnUser(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	userID := chi.URLParam(r, "id")
	deckID := strings.TrimSpace(r.FormValue("deck_id"))
	if err := h.decks.Grant(r.Context(), userID, deckID, "admin", nil); err != nil {
		http.Redirect(w, r, "/admin/users/"+userID+"?err=validation", http.StatusFound)
		return
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "deck_grant", "user", userID, deckID)
	http.Redirect(w, r, "/admin/users/"+userID+"?ok=deck_granted", http.StatusFound)
}
