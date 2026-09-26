//go:build defeito

package chave_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap16/chave"
	"github.com/go-sob-pressao/enxame/examples/cap16/gateway"
)

// Quatro respostas perdidas: cinco tentativas. Quantos débitos?
func TestUUIDDebitaCadaTentativa(t *testing.T) {
	g := &gateway.Gateway{PerderRespostas: 4}
	n := cincoTentativas(t, chave.CobrarComUUID(g))
	if g.Debitos() != 1 {
		t.Fatalf("%d tentativas, %d débitos", n, g.Debitos())
	}
}
