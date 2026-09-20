package storage

import (
	"context"
	"testing"
	"time"
)

func TestMemorySeatFromWinnerID(t *testing.T) {
	s := NewMemory()
	now := time.Now()
	_ = s.SaveMatch(context.Background(), MatchRecord{
		ID: "m", Player1ID: "a", Player2ID: "bot", WinnerID: "bot", StartedAt: now, FinishedAt: now,
	})
	got, err := s.GetMatch(context.Background(), "m")
	if err != nil {
		t.Fatal(err)
	}
	if got.WinnerSeat != 2 {
		t.Fatalf("seat = %d", got.WinnerSeat)
	}
	if got.Result("a") != "loss" {
		t.Fatalf("result = %s", got.Result("a"))
	}
}
