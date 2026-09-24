//go:build !defeito

package ctxsombreado

import (
	"context"
	"errors"
	"testing"
)

func TestUmaFalhaCancelaAsOutras(t *testing.T) {
	if err := BuscarTodos(context.Background(), 10); !errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		t.Fatalf("err=%v", err)
	}
}
