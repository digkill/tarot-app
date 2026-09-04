package mailer

import (
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestBuildMessageEncodesSubject(t *testing.T) {
	msg := string(buildMessage("Tarot", "tarot@sorapure.fun", "user@example.com", "Tarot: подтверждение email", "plain", "<p>html</p>"))
	if !strings.Contains(msg, "From: Tarot <tarot@sorapure.fun>") {
		t.Fatalf("from header: %s", msg[:120])
	}
	if !strings.Contains(msg, "=?UTF-8?q?") && !strings.Contains(msg, "=?utf-8?q?") && !strings.Contains(msg, "=?UTF-8?Q?") {
		t.Fatalf("expected encoded subject, got %s", msg)
	}
	if !strings.Contains(msg, "Content-Type: text/html; charset=UTF-8") {
		t.Fatal("missing html part")
	}
}

func TestComposeVerifyRU(t *testing.T) {
	subject, plain, htmlBody := Compose(KindVerify, "ru", "123456")
	if !strings.Contains(plain, "123456") || !strings.Contains(htmlBody, "123456") {
		t.Fatal("code missing from body")
	}
	if subject == "" {
		t.Fatal("empty subject")
	}
}

func TestLiveBegetSend(t *testing.T) {
	if os.Getenv("SMTP_SMOKE") != "1" {
		t.Skip("set SMTP_SMOKE=1 to send a real message via Beget")
	}
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")
	sender := New(Config{
		Host:     getenv("SMTP_HOST", "smtp.beget.com"),
		Port:     getenv("SMTP_PORT", "465"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     getenv("SMTP_FROM", os.Getenv("SMTP_USERNAME")),
		FromName: getenv("SMTP_FROM_NAME", "Tarot"),
	})
	to := sender.cfg.From
	if to == "" {
		t.Fatal("SMTP_FROM/SMTP_USERNAME missing")
	}
	subject, plain, htmlBody := Compose(KindVerify, "ru", "000000")
	if err := sender.Send(to, subject, plain, htmlBody); err != nil {
		t.Fatalf("beget smtp send: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
