//go:build simulation

package simulation

import (
	"testing"
	"time"
)

// livro:inicio teste-rastro

// A mesma seed tem de produzir a mesma execução, mensagem por mensagem:
// o rastro é um hash de cada entrega (origem, destino, tipo, termo), na
// ordem em que aconteceu. Dez execuções, um rastro só.
func TestMesmaSeedMesmoRastro(t *testing.T) {
	_, _, primeiro := cenarioRaft(42, 10*time.Second)
	for i := range 9 {
		if _, _, r := cenarioRaft(42, 10*time.Second); r != primeiro {
			t.Fatalf(
				"execução %d da seed 42: rastro %x, a primeira deu %x",
				i+2,
				r,
				primeiro,
			)
		}
	}
}

// livro:fim teste-rastro

// livro:inicio teste-rastro-enxame

// O mesmo vale para o Enxame simulado, em vinte seeds: o rastro é um
// hash de cada aquisição de partição, reserva e conclusão, na ordem do
// banco. Três execuções de cada seed, um rastro por seed.
func TestMesmaSeedMesmoRastroEnxame(t *testing.T) {
	for seed := range uint64(20) {
		primeiro := cenarioEnxame(seed + 1).rastro.Sum64()
		for i := range 2 {
			r := cenarioEnxame(seed + 1).rastro.Sum64()
			if r != primeiro {
				t.Fatalf("execução %d da seed %d: rastro %x, a "+
					"primeira deu %x", i+2, seed+1, r, primeiro)
			}
		}
	}
}

// livro:fim teste-rastro-enxame
