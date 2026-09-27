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
