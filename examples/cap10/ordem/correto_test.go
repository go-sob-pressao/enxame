//go:build !defeito

package ordem

import (
	"slices"
	"testing"
)

// livro:inicio ordem-correto

// novoCadastro dá a cada teste o próprio cadastro, com os clientes de
// que ele precisa, e o fecha quando o teste termina.
func novoCadastro(t *testing.T, nomes ...string) *Cadastro {
	t.Helper()
	c := &Cadastro{}
	for _, n := range nomes {
		if err := c.Incluir(t.Context(), n); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(c.Fechar)
	return c
}

func TestIncluirCliente(t *testing.T) {
	t.Parallel()
	c := novoCadastro(t)
	if err := c.Incluir(t.Context(), "ana"); err != nil {
		t.Fatal(err)
	}
}

func TestListarClientes(t *testing.T) {
	t.Parallel()
	c := novoCadastro(t, "ana")
	nomes, err := c.Listar(t.Context())
	if err != nil || !slices.Equal(nomes, []string{"ana"}) {
		t.Fatalf("nomes %v, err %v; esperado [ana]", nomes, err)
	}
}

// livro:fim ordem-correto
