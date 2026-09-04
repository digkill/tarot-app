package usage

import (
	"testing"
	"time"
)

func TestRemaining(t *testing.T) {
	if Remaining(0, 15) != 15 {
		t.Fatal("full remaining")
	}
	if Remaining(15, 15) != 0 {
		t.Fatal("exhausted")
	}
	if Remaining(20, 15) != 0 {
		t.Fatal("over used")
	}
}

func TestCalendarDayMoscow(t *testing.T) {
	loc := LoadLocation("Europe/Moscow")
	// 2026-09-05 01:30 MSK == 2026-09-04 22:30 UTC
	now := time.Date(2026, 9, 4, 22, 30, 0, 0, time.UTC)
	day := CalendarDay(now, loc)
	if got := day.Format("2006-01-02"); got != "2026-09-05" {
		t.Fatalf("moscow day=%s", got)
	}
}

func TestLoadLocationFallback(t *testing.T) {
	loc := LoadLocation("Not/AZone")
	if loc.String() != DefaultTimezone {
		t.Fatalf("fallback=%s", loc.String())
	}
}
