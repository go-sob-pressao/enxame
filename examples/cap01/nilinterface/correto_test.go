//go:build !defeito

package main

import "testing"

func TestNomeValidoDevolveNil(t *testing.T) {
	if err := validar("Ana"); err != nil {
		t.Fatalf("esperava nil, veio %v", err)
	}
	if err := validar(""); err == nil {
		t.Fatal("nome vazio deveria falhar")
	}
}
