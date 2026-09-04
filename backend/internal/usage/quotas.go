package usage

import (
	"strings"
	"time"
)

const (
	FreeDailyCards         = 1
	PremiumDailyCards      = 20
	FreeAdUnlocks          = 5
	FreeInterpretations    = 0
	PremiumInterpretations = 50

	AdMinWatch      = 15 * time.Second
	AdCooldown      = 90 * time.Second
	AdTokenTTL      = 10 * time.Minute
	InterpretGap    = 8 * time.Second
	DefaultTimezone = "Europe/Moscow"
)

func DailyCardLimit(hasPremium bool) int {
	if hasPremium {
		return PremiumDailyCards
	}
	return FreeDailyCards
}

func InterpretationLimit(hasPremium bool) int {
	if hasPremium {
		return PremiumInterpretations
	}
	return FreeInterpretations
}

func AdUnlockLimit(hasPremium bool) int {
	if hasPremium {
		return 0
	}
	return FreeAdUnlocks
}

func Remaining(used, limit int) int {
	if limit <= used {
		return 0
	}
	return limit - used
}

func LoadLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultTimezone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc, _ = time.LoadLocation(DefaultTimezone)
		if loc == nil {
			return time.UTC
		}
		return loc
	}
	return loc
}

func CalendarDay(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}
