package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecutarVersao(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar([]string{"-version"}, &saida, &erros); code != 0 {
		t.Fatalf(
			"código = %d, want 0; stderr: %s",
			code,
			erros.String(),
		)
	}
	if got := strings.TrimSpace(saida.String()); got != "enxamed dev" {
		t.Errorf("saída = %q", got)
	}
}

func TestExecutarSemModoFalhaComMensagemClara(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar(nil, &saida, &erros); code != 2 {
		t.Fatalf("código = %d, want 2", code)
	}
	if !strings.Contains(erros.String(), "-dsn") ||
		!strings.Contains(erros.String(), "-demo") {
		t.Errorf("mensagem não orienta o leitor: %q", erros.String())
	}
}

func TestExecutarFlagInvalida(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar([]string{"-nao-existe"}, &saida, &erros); code != 2 {
		t.Fatalf("código = %d, want 2", code)
	}
}

func TestDemoM0(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar([]string{"-demo", "5"}, &saida, &erros); code != 0 {
		t.Fatalf("código = %d; stderr: %s", code, erros.String())
	}
	if !strings.Contains(saida.String(), "5 jobs, 6 execuções") {
		t.Errorf("saída:\n%s", saida.String())
	}
}
