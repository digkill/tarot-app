package decks

import (
	"path"
	"strings"
)

// CanonicalKeys match bundled RWS filenames without extension (the_fool.jpeg).
var CanonicalKeys = []string{
	"the_fool", "the_magician", "the_high_priestess", "the_empress", "the_emperor",
	"the_hierophant", "the_lovers", "the_chariot", "the_strength", "the_hermit",
	"wheel_of_fortune", "justice", "the_hanged_man", "death", "temperance",
	"the_devil", "the_tower", "the_star", "the_moon", "the_sun", "judgement", "the_world",
	"ace_of_wands", "two_of_wands", "three_of_wands", "four_of_wands", "five_of_wands",
	"six_of_wands", "seven_of_wands", "eight_of_wands", "nine_of_wands", "ten_of_wands",
	"page_of_wands", "knight_of_wands", "queen_of_wands", "king_of_wands",
	"ace_of_cups", "two_of_cups", "three_of_cups", "four_of_cups", "five_of_cups",
	"six_of_cups", "seven_of_cups", "eight_of_cups", "nine_of_cups", "ten_of_cups",
	"page_of_cups", "knight_of_cups", "queen_of_cups", "king_of_cups",
	"ace_of_swords", "two_of_swords", "three_of_swords", "four_of_swords", "five_of_swords",
	"six_of_swords", "seven_of_swords", "eight_of_swords", "nine_of_swords", "ten_of_swords",
	"page_of_swords", "knight_of_swords", "queen_of_swords", "king_of_swords",
	"ace_of_pentacles", "two_of_pentacles", "three_of_pentacles", "four_of_pentacles", "five_of_pentacles",
	"six_of_pentacles", "seven_of_pentacles", "eight_of_pentacles", "nine_of_pentacles", "ten_of_pentacles",
	"page_of_pentacles", "knight_of_pentacles", "queen_of_pentacles", "king_of_pentacles",
}

var canonicalSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(CanonicalKeys))
	for _, k := range CanonicalKeys {
		m[k] = struct{}{}
	}
	return m
}()

const (
	FileBack  = "back"
	FileCover = "cover"
)

// MapFilename maps generator names (00_THE_FOOL.png, CARD_BACK.png) to canonical keys.
func MapFilename(name string) (key string, ok bool) {
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	base = strings.TrimSpace(base)
	if base == "" || strings.HasPrefix(base, ".") {
		return "", false
	}
	ext := strings.ToLower(path.Ext(base))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp":
		base = base[:len(base)-len(ext)]
	default:
		return "", false
	}
	raw := strings.ToUpper(strings.TrimSpace(base))
	raw = strings.ReplaceAll(raw, "-", "_")
	raw = strings.Trim(raw, "_")

	if i := strings.IndexByte(raw, '_'); i >= 0 && i <= 2 {
		allDigits := true
		for _, c := range raw[:i] {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			raw = raw[i+1:]
		}
	}

	switch raw {
	case "CARD_BACK", "BACK", "RUBASHKA", "RUBASHKA_KART":
		return FileBack, true
	case "COVER", "PREVIEW", "CARD_COVER":
		return FileCover, true
	}

	slug := strings.ToLower(raw)
	switch slug {
	case "strength":
		slug = "the_strength"
	case "hanged_man":
		slug = "the_hanged_man"
	}

	if _, exists := canonicalSet[slug]; exists {
		return slug, true
	}
	if _, exists := canonicalSet["the_"+slug]; exists {
		return "the_" + slug, true
	}
	return "", false
}

func IsCardKey(key string) bool {
	_, ok := canonicalSet[key]
	return ok
}

func ProductIDForSlug(slug string) string {
	return "deck_" + slug
}
