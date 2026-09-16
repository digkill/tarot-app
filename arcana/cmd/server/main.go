// Command server runs the Arcana Clash game server.
//
// Environment:
//
//	JWT_SECRET            the main backend's secret (required)
//	DATABASE_URL          Postgres with the main backend's users table; empty
//	                      keeps finished matches in memory (development only)
//	ARCANA_ADDR           listen address, default :$PORT or :8090
//	ARCANA_TURN_SECONDS, ARCANA_MULLIGAN_SECONDS, ARCANA_RECONNECT_SECONDS
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digkill/tarot-app/arcana/internal/api"
	"github.com/digkill/tarot-app/arcana/internal/auth"
	"github.com/digkill/tarot-app/arcana/internal/storage"
	"github.com/digkill/tarot-app/arcana/internal/ws"
)

func main() {
	if err := run(); err != nil {
		slog.Error("arcana: exit", "err", err)
		os.Exit(1)
	}
}

func seconds(name string, def time.Duration) time.Duration {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return def
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	verifier, err := auth.NewVerifier(os.Getenv("JWT_SECRET"))
	if err != nil {
		return err
	}

	var store storage.Store
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := storage.Migrate(ctx, pool); err != nil {
			return err
		}
		store = storage.NewPostgres(pool)
	} else {
		slog.Warn("arcana: DATABASE_URL is empty; finished matches are kept in memory only")
		store = storage.NewMemory()
	}

	cfg := ws.DefaultConfig()
	cfg.Match.TurnTime = seconds("ARCANA_TURN_SECONDS", cfg.Match.TurnTime)
	cfg.Match.MulliganTime = seconds("ARCANA_MULLIGAN_SECONDS", cfg.Match.MulliganTime)
	cfg.Match.ReconnectWindow = seconds("ARCANA_RECONNECT_SECONDS", cfg.Match.ReconnectWindow)
	game := ws.NewServer(cfg, verifier, store)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/arcana/ws", game)
	(&api.API{Verifier: verifier, Store: store, Match: cfg.Match}).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	addr := os.Getenv("ARCANA_ADDR")
	if addr == "" {
		addr = ":8090"
		if port := os.Getenv("PORT"); port != "" {
			addr = ":" + port
		}
	}
	srv := &http.Server{Addr: addr, Handler: api.AcceptStrippedPrefix(mux), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// Live matches cannot survive a restart: record them as server_error.
		game.Shutdown(shutdown)
		_ = srv.Shutdown(shutdown)
	}()
	slog.Info("arcana: listening", "addr", addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
