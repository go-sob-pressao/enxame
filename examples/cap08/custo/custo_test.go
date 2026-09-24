// Package custo — Custo Real #2: Mutex, RWMutex e atomic, com e sem
// contenção.
//
//	go test -bench . -count 10 -cpu 1,8 ./examples/cap08/custo | benchstat -
package custo

import (
	"sync"
	"sync/atomic"
	"testing"
)

// livro:inicio custo-real-2

func BenchmarkMutexSemContencao(b *testing.B) {
	var mu sync.Mutex
	n := 0
	for b.Loop() {
		mu.Lock()
		n++
		mu.Unlock()
	}
}

func BenchmarkMutexComContencao(b *testing.B) {
	var mu sync.Mutex
	n := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			n++
			mu.Unlock()
		}
	})
}

// Leitura pesada: 99 leituras para cada escrita.
func BenchmarkRWMutexLeituraPesada(b *testing.B) {
	var mu sync.RWMutex
	m := map[int]int{1: 1}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%100 == 0 {
				mu.Lock()
				m[1]++
				mu.Unlock()
			} else {
				mu.RLock()
				_ = m[1]
				mu.RUnlock()
			}
			i++
		}
	})
}

func BenchmarkMutexLeituraPesada(b *testing.B) {
	var mu sync.Mutex
	m := map[int]int{1: 1}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			mu.Lock()
			if i%100 == 0 {
				m[1]++
			} else {
				_ = m[1]
			}
			mu.Unlock()
			i++
		}
	})
}

func BenchmarkAtomicComContencao(b *testing.B) {
	var n atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n.Add(1)
		}
	})
}

// livro:fim custo-real-2
