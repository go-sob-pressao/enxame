package chave_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap16/chave"
	"github.com/go-sob-pressao/enxame/examples/cap16/gateway"
)

// livro:inicio experimento-16-1

// Quatro respostas perdidas: cinco tentativas, um débito.
func TestCincoTentativasUmDebito(t *testing.T) {
	g := &gateway.Gateway{PerderRespostas: 4}
	n := cincoTentativas(t, chave.Cobrar(g))
	t.Logf("%d tentativas, %d débito", n, g.Debitos())
	if n != 5 || g.Debitos() != 1 {
		t.Fatalf("%d tentativas, %d débitos", n, g.Debitos())
	}
}

// livro:fim experimento-16-1
