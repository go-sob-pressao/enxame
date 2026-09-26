package lease_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/examples/cap22/lease"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio lease-teste

// A tem o relógio certo; o de B está 15 s adiantado. A toma um lease de
// 10 s. B tenta logo depois.
func TestLeaseComRelogioDeOutroNo(t *testing.T) {
	db := testutil.Postgres(t)
	if err := lease.Criar(t.Context(), db, "lider"); err != nil {
		t.Fatal(err)
	}
	a := lease.No{Nome: "a", Relogio: time.Now, DB: db}
	b := lease.No{Nome: "b", DB: db, Relogio: func() time.Time {
		return time.Now().Add(15 * time.Second)
	}}
	okA, _ := a.Adquirir(t.Context(), "lider", 10*time.Second)
	okB, _ := b.Adquirir(t.Context(), "lider", 10*time.Second)
	t.Logf("relógio de cada nó: A tomou %v, B tomou %v", okA, okB)
	if !okA || !okB {
		t.Fatal("esperava dois donos: para B, o lease de A venceu " +
			"5 s antes de ser concedido")
	}

	if err := lease.Criar(t.Context(), db, "lider2"); err != nil {
		t.Fatal(err)
	}
	okA, _ = a.AdquirirPeloBanco(t.Context(), "lider2", 10*time.Second)
	okB, _ = b.AdquirirPeloBanco(t.Context(), "lider2", 10*time.Second)
	t.Logf("relógio do banco:  A tomou %v, B tomou %v", okA, okB)
	if !okA || okB {
		t.Fatal("com o relógio do banco, só A deveria ser dono")
	}
}

// livro:fim lease-teste
