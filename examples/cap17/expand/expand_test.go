package expand_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap17/expand"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio tres-deploys

// Os três deploys, em ordem, com o código de cada fase rodando ao lado
// do da fase anterior.
func TestTresDeploys(t *testing.T) {
	db := testutil.Postgres(t)
	ctx := t.Context()
	v1, v2, v3 := expand.V1{DB: db}, expand.V2{DB: db}, expand.V3{DB: db}
	passo := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// Antes: só v1, só a coluna valor. 250 pedidos de R$ 10,50.
	passo(expand.Migrar(ctx, db, 1))
	for i := 1; i <= 250; i++ {
		passo(v1.Gravar(ctx, i, 10.50))
	}
	// Deploy 1 — expand. v1 continua no ar; v2 começa a subir.
	passo(expand.Migrar(ctx, db, 2))
	passo(v1.Gravar(ctx, 251, 10.50)) // o trigger preenche os centavos
	passo(v2.Gravar(ctx, 252, 1050))
	n, err := expand.Backfill(ctx, db) // as linhas de antes
	passo(err)
	total, err := v2.Total(ctx)
	passo(err)
	t.Logf("backfill: %d linhas; total: %d centavos", n, total)
	if n != 250 || total != 252*1050 {
		t.Fatalf("backfill %d, total %d", n, total)
	}
	// Deploy 2 — só v2 no ar. Deploy 3 — contract, com v3.
	passo(expand.Migrar(ctx, db, 3))
	passo(v3.Gravar(ctx, 253, 1050))
	if err := v1.Gravar(ctx, 254, 10.50); err == nil {
		t.Fatal("v1 gravou depois do contract")
	} else {
		t.Logf("v1 depois do contract: %v", err)
	}
	// A volta do contract: v2 funciona de novo, sem perder nada.
	passo(expand.Migrar(ctx, db, 2))
	total, err = v2.Total(ctx)
	passo(err)
	if total != 253*1050 {
		t.Fatalf("depois da volta, total %d", total)
	}
}

// livro:fim tres-deploys
