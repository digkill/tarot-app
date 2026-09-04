package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/digkill/tarot-app/backend/internal/decks"
)

func (h *Handler) deckDir(slug string) string {
	return filepath.Join(h.cfg.DeckStorageDir, slug)
}

func (h *Handler) mediaURL(slug, key string) string {
	p := filepath.Join(h.deckDir(slug), key+".jpg")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return "/media/decks/" + slug + "/" + key + ".jpg"
}

func (h *Handler) deckCardURLs(slug string) (cover, back string, cards map[string]string) {
	cards = make(map[string]string)
	for _, key := range decks.CanonicalKeys {
		if u := h.mediaURL(slug, key); u != "" {
			cards[key] = u
		}
	}
	cover = h.mediaURL(slug, decks.FileCover)
	if cover == "" {
		cover = h.mediaURL(slug, "the_fool")
	}
	back = h.mediaURL(slug, decks.FileBack)
	return cover, back, cards
}

func (h *Handler) ServeDeckMedia(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	file := chi.URLParam(r, "file")
	if !validDeckSlug(slug) {
		http.NotFound(w, r)
		return
	}
	safe, ok := decks.SanitizeMediaFile(file)
	if !ok {
		http.NotFound(w, r)
		return
	}
	root, err := filepath.Abs(h.cfg.DeckStorageDir)
	if err != nil {
		http.Error(w, "storage", http.StatusInternalServerError)
		return
	}
	full := filepath.Join(root, slug, safe)
	rel, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.NotFound(w, r)
		return
	}
	if _, err = os.Stat(full); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, full)
}

func validDeckSlug(slug string) bool {
	if len(slug) < 2 || len(slug) > 63 {
		return false
	}
	for i, c := range slug {
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
		if i == 0 && (c == '_' || c == '-') {
			return false
		}
		if !ok {
			return false
		}
	}
	return true
}
