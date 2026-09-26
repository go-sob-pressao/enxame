package retentar_test

import (
	"errors"
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap21/retentar"
)

var errFora = errors.New("destino fora do ar")

// simular roda passos de 1 ms com o destino fora do ar: a cada passo,
// o produtor oferece 3 itens e o consumidor processa 2; os que falham
// voltam ao canal a cada 100 passos — a espera antes de retentar.
func simular(t *testing.T, f *retentar.Fila, passos int) {
	t.Helper()
	aceitos, recusados := 0, 0
	for p := range passos {
		if p%100 == 0 {
			f.Reenviar()
		}
		for range 3 {
			it := retentar.Item{ID: p, Corpo: make([]byte, 1024)}
			if f.Oferecer(it) {
				aceitos++
			} else {
				recusados++
			}
		}
		for range 2 {
			f.Processar(func(retentar.Item) error { return errFora })
		}
		if (p+1)%5000 == 0 {
			c, r := f.Tamanho()
			t.Logf("%5d ms: canal %5d  retentar %5d  "+
				"aceitos %5d  recusados %5d",
				p+1, c, r, aceitos, recusados)
		}
	}
}

// livro:inicio enigma-teste

// O canal nunca passa de 10 mil. A memória, sim.
func TestFilaLimitadaCresce(t *testing.T) {
	f := retentar.Nova(10000)
	simular(t, f, 30000)
	if c, r := f.Tamanho(); c > 10000 || r < 30000 {
		t.Fatalf("canal %d, retentar %d", c, r)
	}
}

// Com o limite no sistema inteiro, retentar para de crescer.
func TestLimiteNoSistema(t *testing.T) {
	f := retentar.Limitada(10000, 10000)
	simular(t, f, 30000)
	if c, r := f.Tamanho(); c+r > 10000 {
		t.Fatalf("canal %d, retentar %d", c, r)
	}
}

// livro:fim enigma-teste
