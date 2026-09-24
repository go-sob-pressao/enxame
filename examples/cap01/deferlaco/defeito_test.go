//go:build defeito

package main

import "testing"

func TestDeferEmLacoSeguraTodos(t *testing.T) {
	p := &Pool{}
	processarTodos(p, 1000)
	if got := p.pico.Load(); got != 1000 {
		t.Fatalf(
			"pico %d; o enigma prevê 1000 abertos ao mesmo tempo",
			got,
		)
	}
}
