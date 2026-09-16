// Command bot plays Arcana Clash matches against whoever joins the queue —
// handy for trying a client alone.
//
//	go run ./cmd/bot -url ws://localhost:8090/api/v1/arcana/ws -secret "$JWT_SECRET"
//
// With -secret the bot signs its own token for a fixed test user id; the
// user must exist when the server stores matches in Postgres.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/arcana/internal/bot"
)

func main() {
	url := flag.String("url", "ws://localhost:8090/api/v1/arcana/ws", "server WebSocket URL")
	token := flag.String("token", "", "access token")
	secret := flag.String("secret", "", "JWT secret to sign a token with instead")
	user := flag.String("user", "00000000-0000-4000-8000-00000000b0b0", "user id for -secret")
	hero := flag.String("hero", "", "hero id (random when empty)")
	loop := flag.Bool("loop", true, "queue again after each match")
	flag.Parse()

	if *token == "" && *secret != "" {
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject: *user, ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		}).SignedString([]byte(*secret))
		if err != nil {
			log.Fatal(err)
		}
		*token = signed
	}
	if *token == "" {
		log.Fatal("-token or -secret is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for {
		b := &bot.Bot{URL: *url, Token: *token, Hero: *hero, Log: log.Printf}
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
