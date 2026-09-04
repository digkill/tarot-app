package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/digkill/tarot-app/backend/internal/atrest"
	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/config"
	"github.com/digkill/tarot-app/backend/internal/httpapi"
	"github.com/digkill/tarot-app/backend/internal/llm"
	"github.com/digkill/tarot-app/backend/internal/mailer"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		slog.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err = goose.SetDialect("postgres"); err != nil {
		slog.Error("goose set dialect", "error", err)
		os.Exit(1)
	}
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("open sql.DB for migrations", "error", err)
		os.Exit(1)
	}
	if err = goose.Up(sqlDB, "migrations"); err != nil {
		slog.Error("run migrations", "error", err)
		_ = sqlDB.Close()
		os.Exit(1)
	}
	_ = sqlDB.Close()

	users := storage.NewUserRepo(pool)
	refreshTokens := storage.NewRefreshTokenRepo(pool)
	readings := storage.NewReadingRepo(pool)
	accessStats := storage.NewAccessStatRepo(pool)
	codes := storage.NewEmailCodeRepo(pool)
	txns := storage.NewTransactionRepo(pool)
	stats := storage.NewStatsRepo(pool)
	audit := storage.NewAuditRepo(pool)
	deckRepo := storage.NewDeckRepo(pool)
	usageRepo := storage.NewUsageRepo(pool)
	checkoutRepo := storage.NewCheckoutRepo(pool)
	if err = os.MkdirAll(cfg.DeckStorageDir, 0o755); err != nil {
		slog.Error("create deck storage", "error", err)
		os.Exit(1)
	}

	if cfg.AdminEmail != "" {
		bootCtx := context.Background()
		existing, getErr := users.GetByEmail(bootCtx, cfg.AdminEmail)
		switch {
		case getErr == nil:
			if existing.Role != storage.RoleAdmin {
				if err = users.SetRole(bootCtx, existing.ID, storage.RoleAdmin); err != nil {
					slog.Error("promote admin user", "error", err)
					os.Exit(1)
				}
			}
			slog.Info("admin role ensured", "email", cfg.AdminEmail)
		case errors.Is(getErr, storage.ErrNotFound):
			if cfg.AdminPassword == "" {
				slog.Warn("ADMIN_EMAIL is not registered and ADMIN_PASSWORD is empty; admin bootstrap skipped")
				break
			}
			hash, hashErr := auth.HashPassword(cfg.AdminPassword)
			if hashErr != nil {
				slog.Error("hash admin password", "error", hashErr)
				os.Exit(1)
			}
			created, ensErr := users.EnsureAdmin(bootCtx, storage.EnsureAdminParams{
				Email:        cfg.AdminEmail,
				PasswordHash: hash,
			})
			if ensErr != nil {
				slog.Error("ensure admin user", "error", ensErr)
				os.Exit(1)
			}
			if created {
				slog.Info("admin user created", "email", cfg.AdminEmail)
			} else {
				slog.Info("admin role ensured", "email", cfg.AdminEmail)
			}
		default:
			slog.Error("lookup admin user", "error", getErr)
			os.Exit(1)
		}
	}

	mail := mailer.New(mailer.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		FromName: cfg.SMTPFromName,
	})

	box, err := atrest.NewBox(cfg.PIIEncryptionKey)
	if err != nil {
		slog.Error("init pii encryption", "error", err)
		os.Exit(1)
	}

	var llmClient llm.Interpreter
	var primary llm.Interpreter
	if cfg.KieAPIKey != "" {
		primary = llm.NewClient(cfg.KieAPIKey, cfg.KieBaseURL, cfg.KieTarotModel, cfg.KieReasoningEffort)
		slog.Info("primary llm enabled")
	} else {
		slog.Warn("KIE_API_KEY is empty; primary AI provider is disabled")
	}
	var fallback llm.Interpreter
	if cfg.OpenAIAPIKey != "" {
		fallback = llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.OpenAITarotModel)
		slog.Info("openai fallback enabled")
	}
	if primary != nil || fallback != nil {
		llmClient = llm.NewService(primary, fallback)
	} else {
		slog.Warn("no AI providers configured; interpretations are disabled")
	}

	handler := httpapi.NewHandler(cfg, users, refreshTokens, readings, accessStats, codes, txns, stats, audit, deckRepo, usageRepo, checkoutRepo, mail, box, llmClient)

	expireStop := make(chan struct{})
	go func() {
		run := func() {
			n, expErr := users.ExpireDue(context.Background())
			if expErr != nil {
				slog.Error("expire premium", "error", expErr)
				return
			}
			if n > 0 {
				slog.Info("expired premium subscriptions", "count", n)
			}
		}
		run()
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-expireStop:
				return
			case <-ticker.C:
				run()
			}
		}
	}()

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler.Router(),
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	close(expireStop)
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
