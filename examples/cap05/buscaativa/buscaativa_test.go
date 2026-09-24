package buscaativa

import (
	"testing"
	"time"
)

func TestBuscaAtivaGiraEnquantoEspera(t *testing.T) {
	c := make(chan int)
	go func() {
		<-time.After(50 * time.Millisecond)
		c <- 42
	}()
	v, voltas := EsperarComDefault(c)
	if v != 42 {
		t.Fatalf("valor %d", v)
	}
	t.Logf("%d voltas de laço em ~50 ms de espera", voltas)
	if voltas < 1000 {
		t.Fatalf("só %d voltas: esperava a busca ativa", voltas)
	}
}
