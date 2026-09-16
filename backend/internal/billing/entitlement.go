package billing

import "strings"

func CanRevokeFrom(currentSource, reporter string) bool {
	current := strings.ToLower(strings.TrimSpace(currentSource))
	reporter = strings.ToLower(strings.TrimSpace(reporter))
	if reporter == "" {
		return false
	}
	if current == "" {
		return reporter == ProviderRuStore || reporter == ProviderDev
	}
	return current == reporter
}

func KeepLifetime(currentProduct, incomingProduct string) bool {
	return strings.TrimSpace(currentProduct) == ProductLifetime && strings.TrimSpace(incomingProduct) != ProductLifetime
}

func IsRuStoreManaged(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case ProviderRuStore, ProviderDev:
		return true
	default:
		return false
	}
}

func IsYooKassaManaged(source string) bool {
	return strings.ToLower(strings.TrimSpace(source)) == ProviderYooKassa
}

// IsAppleManaged reports whether premium came from Apple IAP. Such an
// entitlement may only be changed by a server-verified Apple transaction or
// notification, never by a client self-report.
func IsAppleManaged(source string) bool {
	return strings.ToLower(strings.TrimSpace(source)) == ProviderApple
}

func ActivePremiumSource(hasPremium bool, source string) (string, bool) {
	if !hasPremium {
		return "", false
	}
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == "" {
		return ProviderRuStore, true
	}
	return NormalizeProvider(source), true
}
