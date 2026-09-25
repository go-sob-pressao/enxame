package agrupador

import (
	"context"
	"fmt"
	"sync"
	"testing"
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

// livro:inicio missao-02-teste

// TestMissao: dez eventos seguidos, com lote de dez, formam UM lote.
// Na CI, este teste falha às vezes — às terças, dizem.
func TestMissao(t *testing.T) {
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
	time.Sleep(20 * time.Millisecond) //nolint:forbidigo // o enigma
	if got := c.tamanhos(); len(got) != 1 || got[0] != 10 {
		t.Fatalf("lotes de tamanho %v, esperado [10]", got)
	}
}

// livro:fim missao-02-teste
