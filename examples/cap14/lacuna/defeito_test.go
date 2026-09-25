//go:build defeito

package lacuna_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap14/lacuna"
)

// livro:inicio lacuna-teste

// Duas transações gravam; a que pegou o id menor faz COMMIT por último.
// O leitor passa entre os dois COMMITs.
func TestLeitorPulaEvento(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	leitor := &lacuna.Leitor{}

	t1 := iniciar(t, db)
	if err := lacuna.Gravar(ctx, t1, "a"); err != nil { // id 1
		t.Fatal(err)
	}
	t2 := iniciar(t, db)
	if err := lacuna.Gravar(ctx, t2, "b"); err != nil { // id 2
		t.Fatal(err)
	}
	if err := t2.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	primeira, _ := leitor.Ler(ctx, db) // vê só o id 2
	if err := t1.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	segunda, _ := leitor.Ler(ctx, db) // procura id > 2
	lidos := append(primeira, segunda...)
	if len(lidos) != 2 {
		t.Fatalf("leu %v de [a b]", lidos)
	}
}

// livro:fim lacuna-teste
