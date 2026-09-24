package paralelizar

import (
	"fmt"
	"testing"
)

func dados(n int) []int64 {
	d := make([]int64, n)
	for i := range d {
		d[i] = int64(i % 1000)
	}
	return d
}

func TestMesmoResultadoComQualquerDivisao(t *testing.T) {
	d := dados(100_000)
	esperado := SomarQuadrados(d, 1)
	for _, p := range []int{2, 7, 64, 10_000, 200_000} {
		if got := SomarQuadrados(d, p); got != esperado {
			t.Errorf("partes=%d: %d, esperado %d", p, got, esperado)
		}
	}
}

// livro:inicio bench-partes

// Mesmo trabalho, de 1 a 100 mil goroutines.
func BenchmarkSomarQuadrados(b *testing.B) {
	d := dados(1_000_000)
	for _, partes := range []int{1, 8, 64, 10_000, 100_000} {
		b.Run(fmt.Sprintf("partes=%d", partes), func(b *testing.B) {
			for b.Loop() {
				SomarQuadrados(d, partes)
			}
		})
	}
}

// livro:fim bench-partes
