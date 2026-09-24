package semaforo

import (
	"context"
	"sync/atomic"
	"testing"
)

func tarefas(n int, emExecucao, pico *atomic.Int64) []func() {
	ts := make([]func(), n)
	for i := range ts {
		ts[i] = func() {
			atual := emExecucao.Add(1)
			for v := pico.Load(); atual > v && !pico.CompareAndSwap(v, atual); v = pico.Load() {
				continue
			}
			emExecucao.Add(-1)
		}
	}
	return ts
}

func TestLimite(t *testing.T) {
	var e1, p1, e2, p2 atomic.Int64
	ComCanal(tarefas(500, &e1, &p1), 4)
	if err := ComSemaforo(context.Background(), tarefas(500, &e2, &p2), 4); err != nil {
		t.Fatal(err)
	}
	if p1.Load() > 4 || p2.Load() > 4 {
		t.Fatalf("picos %d e %d; limite 4", p1.Load(), p2.Load())
	}
}
