//go:build defeito

package ctxsombreado

import (
	"context"
	"testing"
	"time"
)

func TestAsOutrasNuncaSaoCanceladas(t *testing.T) {
	fim := make(chan error, 1)
	go func() { fim <- BuscarTodos(context.Background(), 10) }()
	select {
	case err := <-fim:
		t.Fatalf("o enigma não se reproduziu: terminou com %v", err)
	case <-time.After(200 * time.Millisecond):
		t.Log(
			"200 ms depois da primeira falha, as outras nove buscas continuam esperando",
		)
	}
}
