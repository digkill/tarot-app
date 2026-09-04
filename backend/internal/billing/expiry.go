package billing

import (
	"strconv"
	"strings"
	"time"
)

func ExpiresAt(p Product, from time.Time) *time.Time {
	if from.IsZero() {
		from = time.Now()
	}
	switch p.ID {
	case ProductMonthly:
		t := from.AddDate(0, 0, 31)
		return &t
	case ProductYearly:
		t := from.AddDate(0, 0, 366)
		return &t
	case ProductLifetime:
		return nil
	default:
		if p.Kind == KindSubscription {
			t := from.AddDate(0, 0, 31)
			return &t
		}
		return nil
	}
}

func IsStoreManaged(source string) bool {
	return IsRuStoreManaged(source)
}

func ParseTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
		if n > 1_000_000_000_000 {
			t := time.UnixMilli(n)
			return &t
		}
		t := time.Unix(n, 0)
		return &t
	}
	return nil
}

func ChooseExpiry(product Product, fromStore *time.Time, now time.Time) *time.Time {
	if product.ID == ProductLifetime {
		return nil
	}
	if fromStore != nil && !fromStore.IsZero() {
		return fromStore
	}
	return ExpiresAt(product, now)
}
