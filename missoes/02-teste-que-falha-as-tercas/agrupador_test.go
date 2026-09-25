package agrupador

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// coletor guarda os lotes entregues.
type coletor struct {
	mu    sync.Mutex
	lotes [][]string
}

func (c *coletor) entregar(l []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lotes = append(c.lotes, l)
}

func (c *coletor) tamanhos() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	var t []int
	for _, l := range c.lotes {
		t = append(t, len(l))
	}
	return t
}

// intervalo é curto para o teste não demorar.
const intervalo = 300 * time.Microsecond

// Ao sair, o agrupador entrega o lote incompleto.
func TestSaidaEntregaOQueSobrou(t *testing.T) {
	var c coletor
	a := Novo(10, time.Hour, c.entregar)
	ctx, cancel := context.WithCancel(t.Context())
	fim := make(chan struct{})
	go func() {
		a.Rodar(ctx)
		close(fim)
	}()
	for i := range 3 {
		if err := a.Adicionar(ctx, fmt.Sprint(i)); err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	<-fim
	if got := c.tamanhos(); len(got) != 1 || got[0] != 3 {
		t.Fatalf("lotes de tamanho %v, esperado [3]", got)
	}
}

// livro:inicio missao-02-gabarito

// TestMissao: dez eventos seguidos, com lote de dez, formam UM lote.
// Dentro da bolha, o relógio só anda quando todas as goroutines estão
// bloqueadas de vez: enquanto os dez eventos passam de uma goroutine
// para a outra, nenhum tempo passa, e o prazo do lote não pode vencer
// no meio. synctest.Wait espera o agrupador terminar de trabalhar.
func TestMissao(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c coletor
		a := Novo(10, intervalo, c.entregar)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go a.Rodar(ctx)
		for i := range 10 {
			if err := a.Adicionar(ctx, fmt.Sprint(i)); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		if got := c.tamanhos(); len(got) != 1 || got[0] != 10 {
			t.Fatalf("lotes de tamanho %v, esperado [10]", got)
		}
	})
}

// livro:fim missao-02-gabarito
