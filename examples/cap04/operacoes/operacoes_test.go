// Package operacoes prova, por teste, a tabela do Capítulo 4: o que
// cada operação faz num canal aberto, fechado e nil.
package operacoes

import "testing"

func panica(f func()) (ok bool) {
	defer func() { ok = recover() != nil }()
	f()
	return false
}

func TestCanalFechado(t *testing.T) {
	c := make(chan int, 1)
	c <- 7
	close(c)
	if v, ok := <-c; v != 7 || !ok {
		t.Error(
			"receber de fechado ainda entrega o que estava no buffer",
		)
	}
	if v, ok := <-c; v != 0 || ok {
		t.Error("depois do buffer: zero value e ok=false, sem bloquear")
	}
	if !panica(func() { c <- 1 }) {
		t.Error("enviar para fechado deveria entrar em pânico")
	}
	if !panica(func() { close(c) }) {
		t.Error("fechar duas vezes deveria entrar em pânico")
	}
}

func TestCanalNil(t *testing.T) {
	var c chan int
	if !panica(func() { close(c) }) {
		t.Error("fechar nil deveria entrar em pânico")
	}
	select {
	case <-c:
		t.Error("receber de nil nunca deveria ficar pronto")
	case c <- 1:
		t.Error("enviar para nil nunca deveria ficar pronto")
	default: // o único caso possível: nil bloqueia para sempre
	}
}
