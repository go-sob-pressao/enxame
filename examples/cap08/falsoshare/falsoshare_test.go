// Package falsoshare — dois contadores independentes na mesma linha de
// cache.
package falsoshare

import (
	"sync"
	"sync/atomic"
	"testing"
)

// livro:inicio falso-compartilhamento

// Juntos: os dois contadores ocupam 16 bytes vizinhos — a mesma linha
// de 64.
type Juntos struct {
	a, b atomic.Int64
}

// Separados: o preenchimento empurra b para outra linha de cache.
type Separados struct {
	a atomic.Int64
	_ [56]byte
	b atomic.Int64
}

// livro:fim falso-compartilhamento

func martelar(a, b *atomic.Int64, n int) {
	var wg sync.WaitGroup
	wg.Go(func() {
		for range n {
			a.Add(1)
		}
	})
	wg.Go(func() {
		for range n {
			b.Add(1)
		}
	})
	wg.Wait()
}

func BenchmarkJuntos(b *testing.B) {
	var c Juntos
	for b.Loop() {
		martelar(&c.a, &c.b, 100_000)
	}
}

func BenchmarkSeparados(b *testing.B) {
	var c Separados
	for b.Loop() {
		martelar(&c.a, &c.b, 100_000)
	}
}
