// Command bot plays Arcana Clash matches against whoever joins the queue —
// handy for trying a client alone.
//
//	go run ./cmd/bot -url ws://localhost:8090/api/v1/arcana/ws -secret "$JWT_SECRET"
//
// With -secret the bot signs its own token for a fixed test user id. The id
// need not exist in the users table: the server stores such a player as NULL.
//
// Flags fall back to ARCANA_URL, ARCANA_TOKEN, ARCANA_BOT_SECRET,
// ARCANA_BOT_USER and ARCANA_BOT_HERO, for running as a container.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/arcana/internal/bot"
)

func main() {
	url := flag.String("url", env("ARCANA_URL", "ws://localhost:8090/api/v1/arcana/ws"), "server WebSocket URL")
	token := flag.String("token", os.Getenv("ARCANA_TOKEN"), "access token")
	secret := flag.String("secret", os.Getenv("ARCANA_BOT_SECRET"), "JWT secret to sign a token with instead")
	user := flag.String("user", env("ARCANA_BOT_USER", "00000000-0000-4000-8000-00000000b0b0"), "user id for -secret")
	hero := flag.String("hero", os.Getenv("ARCANA_BOT_HERO"), "hero id (random when empty)")
	deck := flag.String("deck", os.Getenv("ARCANA_BOT_DECK"), "art deck slug (classic when empty)")
	loop := flag.Bool("loop", true, "queue again after each match")
	flag.Parse()

	if *token == "" && *secret == "" {
		log.Fatal("-token or -secret is required")
	}
	// A fresh short-lived token per match, so a long-running bot never
	// shows up with an expired one.
	currentToken := func() string {
		if *secret == "" {
			return *token
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject: *user, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}).SignedString([]byte(*secret))
		if err != nil {
			log.Fatal(err)
		}
		return signed
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for {
		b := &bot.Bot{URL: *url, Token: currentToken(), Hero: *hero, Deck: *deck, Log: log.Printf}
		log.Print("waiting for an opponent")
		res, err := b.Play(ctx)
		if err != nil {
			log.Printf("match error: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		} else {
			log.Printf("match %s: %s (%s)", res.MatchID, res.Outcome, res.Reason)
		}
		if !*loop || ctx.Err() != nil {
			return
		}
	}
}

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
