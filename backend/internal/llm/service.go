package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const defaultAttempts = 3

type Service struct {
	primary  Interpreter
	fallback Interpreter
	attempts int
	sleep    func(ctx context.Context, d time.Duration) error
}

func NewService(primary, fallback Interpreter) *Service {
	return &Service{
		primary:  primary,
		fallback: fallback,
		attempts: defaultAttempts,
		sleep:    sleepCtx,
	}
}

func (s *Service) Interpret(ctx context.Context, req InterpretRequest) (Insight, error) {
	attempts := s.attempts
	if attempts < 1 {
		attempts = 1
	}

	var last error
	if s.primary != nil {
		insight, err := s.runAttempts(ctx, s.primary, req, attempts)
		if err == nil {
			return insight, nil
		}
		last = err
		slog.Warn("primary interpretation failed after retries, trying fallback")
	}

	if s.fallback != nil {
		insight, err := s.runAttempts(ctx, s.fallback, req, attempts)
		if err == nil {
			return insight, nil
		}
		last = err
	}

	if last == nil {
		return Insight{}, fmt.Errorf("no interpretation providers configured")
	}
	return Insight{}, last
}

func (s *Service) runAttempts(ctx context.Context, client Interpreter, req InterpretRequest, attempts int) (Insight, error) {
	var last error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return Insight{}, err
		}
		insight, err := client.Interpret(ctx, req)
		if err == nil {
			return insight, nil
		}
		last = err
		slog.Warn("interpretation attempt failed", "attempt", i+1, "of", attempts)
		if i < attempts-1 {
			delay := time.Duration(i+1) * 400 * time.Millisecond
			if sleepErr := s.sleep(ctx, delay); sleepErr != nil {
				return Insight{}, sleepErr
			}
		}
	}
	return Insight{}, last
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
