//go:build !defeito

package main

import "testing"

func TestPonteiroProtege(t *testing.T) {
	if got := somar(1000); got != 1000 {
		t.Fatalf("total %d, esperado 1000", got)
	}
}
