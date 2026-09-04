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
