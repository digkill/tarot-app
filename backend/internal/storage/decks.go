package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digkill/tarot-app/backend/internal/decks"
)

type I18nMap map[string]string

type Deck struct {
	ID               string
	Slug             string
	TitleI18n        I18nMap
	DescriptionI18n  I18nMap
	Theme            decks.Theme
	PriceKop         int
	OriginalPriceKop *int
	Currency         string
	RustoreProductID *string
	IsFree           bool
	IsPublished      bool
	SortOrder        int
	CardCount        int
	HasBack          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (d *Deck) ProductID() string {
	if d.RustoreProductID != nil && strings.TrimSpace(*d.RustoreProductID) != "" {
		return strings.TrimSpace(*d.RustoreProductID)
	}
	return decks.ProductIDForSlug(d.Slug)
}

type DeckRepo struct {
	pool *pgxpool.Pool
}

func NewDeckRepo(pool *pgxpool.Pool) *DeckRepo {
	return &DeckRepo{pool: pool}
}

const deckSelect = `
	id, slug, title_i18n, description_i18n, theme, price_kop, original_price_kop, currency,
	rustore_product_id, is_free, is_published, sort_order, card_count, has_back,
	created_at, updated_at`

func scanDeck(row rowScanner) (*Deck, error) {
	var d Deck
	var titleRaw, descRaw, themeRaw []byte
	err := row.Scan(
		&d.ID, &d.Slug, &titleRaw, &descRaw, &themeRaw, &d.PriceKop, &d.OriginalPriceKop, &d.Currency,
		&d.RustoreProductID, &d.IsFree, &d.IsPublished, &d.SortOrder, &d.CardCount, &d.HasBack,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(titleRaw, &d.TitleI18n)
	_ = json.Unmarshal(descRaw, &d.DescriptionI18n)
	if d.TitleI18n == nil {
		d.TitleI18n = I18nMap{}
	}
	if d.DescriptionI18n == nil {
		d.DescriptionI18n = I18nMap{}
	}
	d.Theme = decks.ThemeFromJSON(themeRaw)
	return &d, nil
}

type UpsertDeckParams struct {
	Slug             string
	TitleI18n        I18nMap
	DescriptionI18n  I18nMap
	Theme            decks.Theme
	PriceKop         int
	OriginalPriceKop *int
	RustoreProductID *string
	IsFree           bool
	IsPublished      bool
	SortOrder        int
}

func (r *DeckRepo) Create(ctx context.Context, p UpsertDeckParams) (*Deck, error) {
	title, _ := json.Marshal(p.TitleI18n)
	desc, _ := json.Marshal(p.DescriptionI18n)
	const q = `
		INSERT INTO decks (
			slug, title_i18n, description_i18n, theme, price_kop, original_price_kop, rustore_product_id,
			is_free, is_published, sort_order
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING ` + deckSelect
	d, err := scanDeck(r.pool.QueryRow(ctx, q,
		p.Slug, title, desc, p.Theme.JSON(), p.PriceKop, p.OriginalPriceKop, p.RustoreProductID,
		p.IsFree, p.IsPublished, p.SortOrder,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert deck: %w", err)
	}
	return d, nil
}

func (r *DeckRepo) Update(ctx context.Context, id string, p UpsertDeckParams) (*Deck, error) {
	title, _ := json.Marshal(p.TitleI18n)
	desc, _ := json.Marshal(p.DescriptionI18n)
	const q = `
		UPDATE decks SET
			title_i18n = $2,
			description_i18n = $3,
			theme = $4,
			price_kop = $5,
			original_price_kop = $6,
			rustore_product_id = $7,
			is_free = $8,
			is_published = $9,
			sort_order = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING ` + deckSelect
	d, err := scanDeck(r.pool.QueryRow(ctx, q,
		id, title, desc, p.Theme.JSON(), p.PriceKop, p.OriginalPriceKop, p.RustoreProductID,
		p.IsFree, p.IsPublished, p.SortOrder,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("update deck: %w", err)
	}
	return d, nil
}

func (r *DeckRepo) SetImportMeta(ctx context.Context, id string, cards int, hasBack bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE decks SET card_count = $2, has_back = $3, updated_at = NOW() WHERE id = $1`,
		id, cards, hasBack)
	if err != nil {
		return fmt.Errorf("set import meta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DeckRepo) GetByID(ctx context.Context, id string) (*Deck, error) {
	d, err := scanDeck(r.pool.QueryRow(ctx, `SELECT `+deckSelect+` FROM decks WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get deck: %w", err)
	}
	return d, nil
}

func (r *DeckRepo) GetBySlug(ctx context.Context, slug string) (*Deck, error) {
	d, err := scanDeck(r.pool.QueryRow(ctx, `SELECT `+deckSelect+` FROM decks WHERE slug = $1`, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get deck by slug: %w", err)
	}
	return d, nil
}

func (r *DeckRepo) GetByProductID(ctx context.Context, productID string) (*Deck, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, ErrNotFound
	}
	const q = `
		SELECT ` + deckSelect + ` FROM decks
		WHERE rustore_product_id = $1
		   OR ('deck_' || slug) = $1`
	d, err := scanDeck(r.pool.QueryRow(ctx, q, productID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get deck by product: %w", err)
	}
	return d, nil
}

func (r *DeckRepo) ListAll(ctx context.Context) ([]Deck, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+deckSelect+` FROM decks ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, fmt.Errorf("list decks: %w", err)
	}
	defer rows.Close()
	out := make([]Deck, 0)
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DeckRepo) ListPublished(ctx context.Context) ([]Deck, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+deckSelect+` FROM decks
		WHERE is_published = TRUE
		ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, fmt.Errorf("list published decks: %w", err)
	}
	defer rows.Close()
	out := make([]Deck, 0)
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DeckRepo) Grant(ctx context.Context, userID, deckID, source string, txID *string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_decks (user_id, deck_id, source, transaction_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, deck_id) DO NOTHING`, userID, deckID, source, txID)
	if err != nil {
		return fmt.Errorf("grant deck: %w", err)
	}
	return nil
}

func (r *DeckRepo) Revoke(ctx context.Context, userID, deckID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM user_decks WHERE user_id = $1 AND deck_id = $2`, userID, deckID)
	if err != nil {
		return fmt.Errorf("revoke deck: %w", err)
	}
	return nil
}

func (r *DeckRepo) OwnedIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT deck_id FROM user_decks WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list owned decks: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *DeckRepo) OwnedSlugs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.slug FROM user_decks ud
		JOIN decks d ON d.id = ud.deck_id
		WHERE ud.user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list owned slugs: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var slug string
		if err = rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, rows.Err()
}

func (r *DeckRepo) Owns(ctx context.Context, userID, deckID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_decks WHERE user_id = $1 AND deck_id = $2)`,
		userID, deckID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("owns deck: %w", err)
	}
	return ok, nil
}
