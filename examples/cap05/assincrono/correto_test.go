//go:build !defeito

package main

import "testing"

func TestProcessamentoSobreviveAoHandler(t *testing.T) {
	if err := resultado(t); err != nil {
		t.Fatalf("processamento falhou: %v", err)
	}
}
