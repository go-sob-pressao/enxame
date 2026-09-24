//go:build !defeito

package main

import (
	"errors"
	"testing"
)

func TestFalhaChegaAoChamador(t *testing.T) {
	_, err := somarEstoque([]int{1, 2, 3, 4})
	if !errors.Is(err, ErrIndisponivel) {
		t.Fatalf("esperava ErrIndisponivel, veio %v", err)
	}
}
