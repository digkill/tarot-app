package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/atrest"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	PIIEncryptionKey   []byte
	SMTPHost           string
	SMTPPort           string
	SMTPUsername       string
	SMTPPassword       string
	SMTPFrom           string
	SMTPFromName       string
	KieAPIKey          string
	KieBaseURL         string
	KieTarotModel      string
	KieReasoningEffort string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	CORSAllowedOrigins []string
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
	}

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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
