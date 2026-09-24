package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecutarVersao(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar([]string{"version"}, &saida, &erros); code != 0 {
		t.Fatalf("código = %d, want 0", code)
	}
	if got := strings.TrimSpace(saida.String()); got != "enxamectl dev" {
		t.Errorf("saída = %q", got)
	}
}

func TestExecutarComandoAindaNaoConstruido(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar([]string{"job", "insert"}, &saida, &erros); code != 1 {
		t.Fatalf("código = %d, want 1", code)
	}
	if !strings.Contains(erros.String(), "Capítulo 19") {
		t.Errorf("mensagem não aponta o capítulo: %q", erros.String())
	}
}
