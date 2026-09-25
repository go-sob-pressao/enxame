//go:build defeito

package trava

import (
	"testing"
	"testing/synctest"
	"time"
)

// livro:inicio trava-defeito

// A renovação segura a trava durante o Sleep; a leitura espera a
// trava. Dentro da bolha, esperar um Mutex não é bloqueio durável, e o
// relógio da bolha nunca anda: o teste não termina. Rode com -timeout.
func TestLeituraDuranteRenovacao(t *testing.T) {
	if testing.Short() {
		t.Skip("trava de propósito: rode sem -short, com -timeout")
	}
	synctest.Test(t, func(t *testing.T) {
		var c Cache
		go c.RenovarLento(func() string { return "novo" })
		synctest.Wait() // a renovação pegou a trava e está no Sleep
		lido := make(chan string)
		go func() { lido <- c.Valor() }() // espera a trava
		select {
		case v := <-lido:
			t.Logf("leu %q", v)
		case <-time.After(5 * time.Second):
			t.Fatal("a leitura não voltou em 5 s")
		}
	})
}

// livro:fim trava-defeito
