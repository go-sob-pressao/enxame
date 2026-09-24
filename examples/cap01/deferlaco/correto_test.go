//go:build !defeito

package main

import "testing"

func TestUmAbertoPorVez(t *testing.T) {
	p := &Pool{}
	processarTodos(p, 1000)
	if got := p.pico.Load(); got != 1 {
		t.Fatalf("pico %d, esperado 1", got)
	}
}
