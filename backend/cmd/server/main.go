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

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/digkill/tarot-app/backend/internal/config"
	"github.com/digkill/tarot-app/backend/internal/httpapi"
	"github.com/digkill/tarot-app/backend/internal/llm"
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

	var llmClient *llm.Client
	if cfg.KieAPIKey != "" {
		llmClient = llm.NewClient(cfg.KieAPIKey, cfg.KieBaseURL, cfg.KieTarotModel, cfg.KieReasoningEffort)
		slog.Info("kie llm enabled", "model", cfg.KieTarotModel, "effort", cfg.KieReasoningEffort)
	} else {
		slog.Warn("KIE_API_KEY is empty; AI interpretations are disabled")
	}

	handler := httpapi.NewHandler(cfg, users, refreshTokens, readings, llmClient)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 90 * time.Second,
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
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
