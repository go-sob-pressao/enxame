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

func TestExecutarSemPapeisFalhaComMensagemClara(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar(nil, &saida, &erros); code != 1 {
		t.Fatalf("código = %d, want 1", code)
	}
	if !strings.Contains(erros.String(), "Capítulo 2") {
		t.Errorf("mensagem não aponta o capítulo: %q", erros.String())
	}
}

func TestExecutarFlagInvalida(t *testing.T) {
	var saida, erros bytes.Buffer
	if code := executar([]string{"-nao-existe"}, &saida, &erros); code != 2 {
		t.Fatalf("código = %d, want 2", code)
	}
}
