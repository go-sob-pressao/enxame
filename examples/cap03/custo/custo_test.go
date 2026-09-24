// Package custo mede o que o Capítulo 3 chama de custo real de uma
// goroutine.
//
//	go test -bench . -benchmem -count 10 ./examples/cap03/custo | tee novo.txt
//	benchstat novo.txt
package custo

import (
	"runtime"
	"sync"
	"testing"
)

// livro:inicio custo-goroutine

// Criar uma goroutine, rodá-la até o fim e esperar por ela.
func BenchmarkCriarEEsperar(b *testing.B) {
	var wg sync.WaitGroup
	for b.Loop() {
		wg.Go(func() {})
		wg.Wait()
	}
}

// Troca de contexto: duas goroutines se revezando por canais sem
// buffer. Cada iteração é uma ida e uma volta.
func BenchmarkTrocaDeContexto(b *testing.B) {
	ping, pong := make(chan struct{}), make(chan struct{})
	go func() {
		for range ping {
			pong <- struct{}{}
		}
		close(pong)
	}()
	for b.Loop() {
		ping <- struct{}{}
		<-pong
	}
	close(ping)
}

// Memória: a pilha inicial de cada goroutine bloqueada.
func BenchmarkMemoriaPorGoroutine(b *testing.B) {
	const n = 10_000
	for b.Loop() {
		var antes, depois runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&antes)
		bloqueio := make(chan struct{})
		var wg sync.WaitGroup
		for range n {
			wg.Go(func() { <-bloqueio })
		}
		runtime.ReadMemStats(&depois)
		b.ReportMetric(
			float64(depois.StackInuse-antes.StackInuse)/n,
			"bytes-de-pilha/goroutine",
		)
		close(bloqueio)
		wg.Wait()
	}
}

// livro:fim custo-goroutine
