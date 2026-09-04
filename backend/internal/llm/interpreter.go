package llm

import (
	"context"
)

type Interpreter interface {
	Interpret(ctx context.Context, req InterpretRequest) (Insight, error)
}
