package config

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/atrest"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	JWTSecret              string
	PIIEncryptionKey       []byte
	SMTPHost               string
	SMTPPort               string
	SMTPUsername           string
	SMTPPassword           string
	SMTPFrom               string
	SMTPFromName           string
	KieAPIKey              string
	KieBaseURL             string
	KieTarotModel          string
	KieReasoningEffort     string
	OpenAIAPIKey           string
	OpenAIBaseURL          string
	OpenAITarotModel       string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	CORSAllowedOrigins     []string
	AdminEmail             string
	AdminPassword          string
	AdminSessionTTL        time.Duration
	DeckStorageDir         string
	PublicBaseURL          string
	YooKassaShopID         string
	YooKassaSecretKey      string
	YooKassaVatCode        int
	CloudPaymentsPublicID  string
	CloudPaymentsAPISecret string
	IsProd                 bool

	// Apple IAP. Deliberately outside the IS_PROD/_TEST convention: the same
	// credentials serve both Apple environments, and Apple marks each
	// transaction's environment itself.
	AppleBundleID      string
	AppleAppAppleID    int64
	AppleIAPIssuerID   string
	AppleIAPKeyID      string
	AppleIAPPrivateKey *ecdsa.PrivateKey
	AppleAllowSandbox  bool
	AppleRootCAFile    string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		SMTPHost:           getEnv("SMTP_HOST", "smtp.beget.com"),
		SMTPPort:           getEnv("SMTP_PORT", "465"),
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:           getEnv("SMTP_FROM", os.Getenv("SMTP_USERNAME")),
		SMTPFromName:       getEnv("SMTP_FROM_NAME", "Tarot"),
		KieAPIKey:          os.Getenv("KIE_API_KEY"),
		KieBaseURL:         getEnv("KIE_BASE_URL", "https://api.kie.ai"),
		KieTarotModel:      getEnv("KIE_TAROT_MODEL", "gpt-5-6-luna"),
		KieReasoningEffort: getEnv("KIE_REASONING_EFFORT", "low"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:      getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
		OpenAITarotModel:   getEnv("OPENAI_TAROT_MODEL", "gpt-4o-mini"),
		AdminEmail:         strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		DeckStorageDir:     getEnv("DECK_STORAGE_DIR", "data/decks"),
		PublicBaseURL:      strings.TrimRight(getEnv("PUBLIC_BASE_URL", "https://tarot.sorapure.fun"), "/"),
		YooKassaVatCode:    getEnvInt("YOOKASSA_VAT_CODE", 1),
		IsProd:             getEnvBool("IS_PROD", true),
	}

	envSuffix := ""
	if !cfg.IsProd {
		envSuffix = "_TEST"
	}
	cfg.YooKassaShopID = strings.TrimSpace(os.Getenv("YOOKASSA_SHOP_ID" + envSuffix))
	cfg.YooKassaSecretKey = strings.TrimSpace(os.Getenv("YOOKASSA_SECRET_KEY" + envSuffix))
	cfg.CloudPaymentsPublicID = strings.TrimSpace(os.Getenv("CLOUDPAYMENTS_PUBLIC_ID" + envSuffix))
	cfg.CloudPaymentsAPISecret = strings.TrimSpace(os.Getenv("CLOUDPAYMENTS_API_SECRET" + envSuffix))

	// No envSuffix for Apple: one key serves both Apple environments, and each
	// transaction carries its own environment claim.
	cfg.AppleBundleID = strings.TrimSpace(getEnv("APPLE_BUNDLE_ID", "org.mediarise.tarot"))
	cfg.AppleAppAppleID = int64(getEnvInt("APPLE_APP_APPLE_ID", 0))
	cfg.AppleIAPIssuerID = strings.TrimSpace(os.Getenv("APPLE_IAP_ISSUER_ID"))
	cfg.AppleIAPKeyID = strings.TrimSpace(os.Getenv("APPLE_IAP_KEY_ID"))
	cfg.AppleAllowSandbox = getEnvBool("APPLE_ALLOW_SANDBOX", true)
	cfg.AppleRootCAFile = strings.TrimSpace(os.Getenv("APPLE_ROOT_CA_FILE"))

	appleKey, appleKeyErr := parseAppleP8(os.Getenv("APPLE_IAP_PRIVATE_KEY"), os.Getenv("APPLE_IAP_PRIVATE_KEY_FILE"))
	if appleKeyErr != nil {
		return nil, appleKeyErr
	}
	cfg.AppleIAPPrivateKey = appleKey

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	var err error
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	cfg.PIIEncryptionKey, err = atrest.ParseKey(os.Getenv("PII_ENCRYPTION_KEY"))
	if err != nil {
		return nil, err
	}
	if cfg.SMTPFrom == "" {
		cfg.SMTPFrom = cfg.SMTPUsername
	}
	if cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
		return nil, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD are required")
	}

	switch cfg.KieReasoningEffort {
	case "low", "medium", "high", "xhigh":
	default:
		return nil, fmt.Errorf("KIE_REASONING_EFFORT must be one of low, medium, high, xhigh")
	}

	for _, o := range strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, o)
		}
	}

	cfg.AccessTokenTTL, err = parseDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.RefreshTokenTTL, err = parseDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return nil, err
	}
	cfg.AdminSessionTTL, err = parseDuration("ADMIN_SESSION_TTL", 12*time.Hour)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// parseAppleP8 loads an App Store Connect In-App Purchase private key. It
// accepts a file path, a raw PEM (including one with literal \n escapes, as
// pasted into a docker env var) or base64 of the whole .p8 file.
//
// A blank value returns (nil, nil): like every other payment secret, absence
// simply disables the provider. A value that is present but unparseable is a
// startup error, so a typo surfaces on deploy rather than at the first purchase.
func parseAppleP8(raw, file string) (*ecdsa.PrivateKey, error) {
	raw = strings.TrimSpace(raw)
	file = strings.TrimSpace(file)

	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("APPLE_IAP_PRIVATE_KEY_FILE: %w", err)
		}
		raw = string(b)
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	der := []byte(strings.ReplaceAll(raw, `\n`, "\n"))
	if !strings.Contains(string(der), "-----BEGIN") {
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(raw), ""))
		if err != nil {
			return nil, fmt.Errorf("APPLE_IAP_PRIVATE_KEY: not a PEM block and not valid base64: %w", err)
		}
		der = decoded
	}
	if strings.Contains(string(der), "-----BEGIN") {
		block, _ := pem.Decode(der)
		if block == nil {
			return nil, fmt.Errorf("APPLE_IAP_PRIVATE_KEY: no PEM block found")
		}
		der = block.Bytes
	}

	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		// Older keys may be raw SEC 1 rather than PKCS#8.
		if ecKey, ecErr := x509.ParseECPrivateKey(der); ecErr == nil {
			return ecKey, nil
		}
		return nil, fmt.Errorf("APPLE_IAP_PRIVATE_KEY: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("APPLE_IAP_PRIVATE_KEY: expected an ECDSA key, got %T", parsed)
	}
	return key, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q: %w", key, v, err)
	}
	return d, nil
}
