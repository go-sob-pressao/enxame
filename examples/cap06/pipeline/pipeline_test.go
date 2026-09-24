package pipeline

import (
	"context"
	"errors"
	"testing"
)

func par(v int) bool { return v%2 == 0 }

func TestPipelineCompleto(t *testing.T) {
	ctx := context.Background()
	total, err := Somar(ctx, Filtrar(ctx, Gerar(ctx, 100), par))
	if err != nil || total != 2550 {
		t.Fatalf("total=%d err=%v", total, err)
	}
}

func TestPipelineCancelado(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entrada := Filtrar(ctx, Gerar(ctx, 1_000_000), par)
	<-entrada
	cancel()
	if _, err := Somar(ctx, entrada); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("err=%v", err)
	}
}
