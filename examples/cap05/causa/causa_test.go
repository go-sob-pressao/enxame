package causa

import (
	"context"
	"testing"
	"time"
)

func TestClassificar(t *testing.T) {
	expirado, c1 := context.WithTimeout(
		context.Background(),
		time.Nanosecond,
	)
	defer c1()
	<-expirado.Done()

	desligando, c2 := context.WithCancelCause(context.Background())
	c2(ErrDesligamento)

	cancelado, c3 := context.WithCancel(context.Background())
	c3()

	casos := []struct {
		nome string
		ctx  context.Context
		quer Decisao
	}{
		{"prazo estourado", expirado, ContarFalha},
		{"desligamento", desligando, Repetir},
		{"cancelado pelo cliente", cancelado, Desistir},
	}
	for _, c := range casos {
		if got := Classificar(c.ctx, c.ctx.Err()); got != c.quer {
			t.Errorf("%s: %s, esperado %s", c.nome, got, c.quer)
		}
	}
}
