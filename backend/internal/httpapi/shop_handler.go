package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/digkill/tarot-app/backend/internal/decks"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

type shopDeckView struct {
	Slug             string            `json:"slug"`
	Titles           map[string]string `json:"titles"`
	Descriptions     map[string]string `json:"descriptions"`
	Theme            decks.Theme       `json:"theme"`
	PriceKop         int               `json:"priceKop"`
	OriginalPriceKop *int              `json:"originalPriceKop,omitempty"`
	ProductID        string            `json:"productId"`
	// AppleProductID is the StoreKit SKU the iOS client must request. Absent
	// when the deck is not sold on iOS, and the client must then not offer it.
	AppleProductID string            `json:"appleProductId,omitempty"`
	IsFree         bool              `json:"isFree"`
	Owned          bool              `json:"owned"`
	CardCount      int               `json:"cardCount"`
	HasBack        bool              `json:"hasBack"`
	CoverURL       string            `json:"coverUrl"`
	BackURL        string            `json:"backUrl"`
	Cards          map[string]string `json:"cards"`
}

func (h *Handler) toShopView(d *storage.Deck, owned bool) shopDeckView {
	cover, back, cards := h.deckCardURLs(d.Slug)
	return shopDeckView{
		Slug:             d.Slug,
		Titles:           d.TitleI18n,
		Descriptions:     d.DescriptionI18n,
		Theme:            d.Theme,
		PriceKop:         d.PriceKop,
		OriginalPriceKop: d.OriginalPriceKop,
		ProductID:        d.ProductID(),
		AppleProductID:   strOrEmpty(d.AppleProductID),
		IsFree:           d.IsFree,
		Owned:            owned || d.IsFree,
		CardCount:        d.CardCount,
		HasBack:          d.HasBack,
		CoverURL:         cover,
		BackURL:          back,
		Cards:            cards,
	}
}

// strOrEmpty is the JSON-facing counterpart of derefString, which renders a
// dash for templates.
func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *Handler) ownedSet(r *http.Request) map[string]struct{} {
	out := map[string]struct{}{}
	userID := userIDFromCtx(r.Context())
	if userID == "" {
		return out
	}
	ids, err := h.decks.OwnedIDs(r.Context(), userID)
	if err != nil {
		return out
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func (h *Handler) ShopListDecks(w http.ResponseWriter, r *http.Request) {
	list, err := h.decks.ListPublished(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list decks")
		return
	}
	owned := h.ownedSet(r)
	views := make([]shopDeckView, 0, len(list))
	for i := range list {
		d := list[i]
		_, has := owned[d.ID]
		views = append(views, h.toShopView(&d, has))
	}
	writeJSON(w, http.StatusOK, map[string]any{"decks": views})
}

func (h *Handler) ShopGetDeck(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	d, err := h.decks.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "deck not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load deck")
		return
	}
	if !d.IsPublished {
		writeError(w, http.StatusNotFound, "not_found", "deck not found")
		return
	}
	owned := false
	if userID := userIDFromCtx(r.Context()); userID != "" {
		owned, _ = h.decks.Owns(r.Context(), userID, d.ID)
	}
	writeJSON(w, http.StatusOK, h.toShopView(d, owned))
}

func (h *Handler) MyDecks(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	slugs, err := h.decks.OwnedSlugs(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list owned decks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"slugs": slugs})
}
