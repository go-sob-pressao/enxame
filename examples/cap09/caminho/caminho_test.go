package caminho

import (
	"sync/atomic"
	"testing"
)

func destinos(n int) []string {
	d := make([]string, n)
	for i := range d {
		d[i] = "cliente"
	}
	return d
}

// livro:inicio caminho-testes

// O único teste que a equipe tinha: tudo dá certo na primeira
// tentativa. Com -race, passa — o detector só vê o código que o teste
// executa.
func TestCaminhoFeliz(t *testing.T) {
	var e Enviador
	e.EnviarTodos(destinos(50), func(string) error { return nil })
	if e.entregues != 50 {
		t.Fatalf("entregues %d", e.entregues)
	}
}

// O teste que faltava: uma chamada em cada duas falha. Com -race, na
// versão com defeito, o detector acusa a escrita em repeticoes.
func TestCaminhoDeRetry(t *testing.T) {
	var e Enviador
	var chamadas atomic.Int64
	e.EnviarTodos(destinos(50), func(string) error {
		if chamadas.Add(1)%2 == 0 {
			return ErrTransitorio
		}
		return nil
	})
	if e.entregues != 50 {
		t.Fatalf("entregues %d", e.entregues)
	}
}

// livro:fim caminho-testes
