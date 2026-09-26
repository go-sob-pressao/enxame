package main

import (
	"bytes"
	"strings"
	"testing"
)

func semAmbiente(string) string { return "" }

func TestExecutarVersao(t *testing.T) {
	var saida, erros bytes.Buffer
	code := executar([]string{"version"}, &saida, &erros, semAmbiente)
	if code != 0 {
		t.Fatalf("código = %d, want 0", code)
	}
	if got := strings.TrimSpace(saida.String()); got != "enxamectl dev" {
		t.Errorf("saída = %q", got)
	}
}

func TestComandoDesconhecido(t *testing.T) {
	var saida, erros bytes.Buffer
	code := executar([]string{"job", "voar"}, &saida, &erros, semAmbiente)
	if code != 2 {
		t.Fatalf("código = %d, want 2", code)
	}
}

func TestFlagObrigatoria(t *testing.T) {
	var saida, erros bytes.Buffer
	code := executar([]string{"job", "insert", "--queue", "q"},
		&saida, &erros, semAmbiente)
	if code != 1 || !strings.Contains(erros.String(), "--kind") {
		t.Fatalf("código %d, erro %q", code, erros.String())
	}
}
