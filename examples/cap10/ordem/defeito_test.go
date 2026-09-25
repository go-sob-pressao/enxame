//go:build defeito

package ordem

import (
	"context"
	"slices"
	"testing"
)

// livro:inicio ordem-defeito

// cadastro é compartilhado por todos os testes do pacote.
var cadastro = &Cadastro{}

func TestIncluirCliente(t *testing.T) {
	if err := cadastro.Incluir(context.Background(), "ana"); err != nil {
		t.Fatal(err)
	}
}

// Passa porque, na ordem do arquivo, TestIncluirCliente rodou antes.
func TestListarClientes(t *testing.T) {
	nomes, err := cadastro.Listar(context.Background())
	if err != nil || !slices.Equal(nomes, []string{"ana"}) {
		t.Fatalf("nomes %v, err %v; esperado [ana]", nomes, err)
	}
}

// livro:fim ordem-defeito
