package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubInterpreter struct {
	failTimes int
	calls     *int
	insight   Insight
	err       error
}

func (s *stubInterpreter) Interpret(ctx context.Context, _ InterpretRequest) (Insight, error) {
	if s.calls != nil {
		*s.calls++
	}
	if err := ctx.Err(); err != nil {
		return Insight{}, err
	}
	if s.failTimes > 0 {
		s.failTimes--
		if s.err != nil {
			return Insight{}, s.err
		}
		return Insight{}, errors.New("stub fail")
	}
	return s.insight, nil
}

func TestServiceRetriesPrimaryThenFallback(t *testing.T) {
	var primaryCalls, fallbackCalls int
	primary := &stubInterpreter{
		failTimes: 3,
		calls:     &primaryCalls,
		err:       errors.New("primary down"),
	}
	fallback := &stubInterpreter{
		failTimes: 0,
		calls:     &fallbackCalls,
		insight:   Insight{Summary: "from fallback"},
	}
	svc := NewService(primary, fallback)
	svc.sleep = func(context.Context, time.Duration) error { return nil }

	insight, err := svc.Interpret(context.Background(), InterpretRequest{SpreadName: "test"})
	if err != nil {
		t.Fatalf("Interpret: %v", err)
	}
	if insight.Summary != "from fallback" {
		t.Fatalf("summary: %q", insight.Summary)
	}
	if primaryCalls != 3 {
		t.Fatalf("primary calls: %d", primaryCalls)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback calls: %d", fallbackCalls)
	}
}

func TestServiceSucceedsOnThirdPrimaryAttempt(t *testing.T) {
	var primaryCalls int
	primary := &stubInterpreter{
		failTimes: 2,
		calls:     &primaryCalls,
		insight:   Insight{Summary: "ok"},
	}
	svc := NewService(primary, nil)
	svc.sleep = func(context.Context, time.Duration) error { return nil }

	insight, err := svc.Interpret(context.Background(), InterpretRequest{})
	if err != nil {
		t.Fatalf("Interpret: %v", err)
	}
	if insight.Summary != "ok" {
		t.Fatalf("summary: %q", insight.Summary)
	}
	if primaryCalls != 3 {
		t.Fatalf("primary calls: %d", primaryCalls)
	}
}

func TestParseOpenAIOutput(t *testing.T) {
	raw := []byte(`{"choices":[{"message":{"content":"{\"summary\":\"ok\",\"positions\":[]}"}}]}`)
	text, err := parseOpenAIOutput(200, raw)
	if err != nil {
		t.Fatalf("parseOpenAIOutput: %v", err)
	}
	if text != `{"summary":"ok","positions":[]}` {
		t.Fatalf("text: %q", text)
	}
}
