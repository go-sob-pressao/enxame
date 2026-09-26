//go:build defeito

package versao_test

import (
	"testing"
	"time"
)

// livro:inicio versao-teste

// Um pedido de ontem dorme esperando a entrega; o deploy de hoje sobe;
// a entrega chega, e o run é reexecutado com o código novo.
func TestRunAntigoCodigoNovo(t *testing.T) {
	c := novo()
	id := c.iniciar(t)
	if _, err := c.avancar(t, c.f.Antes, id); err != nil {
		t.Fatal(err)
	}
	c.agora = c.agora.Add(49 * time.Hour)
	if _, err := c.avancar(t, c.f.Depois, id); err != nil {
		t.Fatal(err)
	}
}

// livro:fim versao-teste
