package id_test

import (
	"fmt"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/core/id"
)

// A função é estável: estes valores não podem mudar nunca (ADR-010).
// Os esperados foram calculados fora do Go, com o FNV-1a da definição.
func TestParticaoEstavel(t *testing.T) {
	for chave, quer := range map[string]int{"abc": 267} {
		if got := id.Particao(chave); got != quer {
			t.Fatalf("Particao(%q) = %d, quer %d: a função mudou",
				chave, got, quer)
		}
	}
}

// Chaves diferentes se espalham: em 100 mil, nenhuma partição passa de
// 1,5 vez a média.
func TestParticaoEspalha(t *testing.T) {
	cont := make([]int, id.NumParticoes)
	const n = 100_000
	for i := range n {
		cont[id.Particao(fmt.Sprintf("job-%d", i))]++
	}
	media := n / id.NumParticoes
	for p, c := range cont {
		if c > media*3/2 || c < media/2 {
			t.Fatalf("partição %d com %d chaves; média %d", p, c, media)
		}
	}
}
