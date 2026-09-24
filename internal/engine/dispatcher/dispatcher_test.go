package dispatcher_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/engine/dispatcher"
)

type pool func(ctx context.Context) error

func (p pool) Run(ctx context.Context) error { return p(ctx) }

func TestUmPoolFalhaTodosParam(t *testing.T) {
	falha := errors.New("fila de pagamentos sem banco")
	d := &dispatcher.Dispatcher{Pools: map[string]dispatcher.Runner{
		"emails": pool(
			func(ctx context.Context) error { <-ctx.Done(); return nil },
		),
		"pagamentos": pool(
			func(context.Context) error { return falha },
		),
	}}
	if err := d.Run(context.Background()); !errors.Is(err, falha) {
		t.Fatalf("Run devolveu %v", err)
	}
}
