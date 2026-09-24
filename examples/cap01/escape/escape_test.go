package main

import "testing"

// sumidouro impede que o compilador descarte o resultado e, com ele, a
// alocação.
var sumidouro any

// As alocações por chamada confirmam o que -gcflags=-m anuncia.
func TestAlocacoes(t *testing.T) {
	casos := []struct {
		nome string
		fn   func()
		quer float64
	}{
		{"naPilha", func() { p := naPilha(); sumidouro = p.X }, 0},
		{"naHeap", func() { sumidouro = naHeap(len(t.Name()), 4) }, 1},
		{
			"viaInterface",
			func() { sumidouro = viaInterface(len(t.Name()), 6) },
			1,
		},
	}
	for _, c := range casos {
		if got := testing.AllocsPerRun(100, c.fn); got != c.quer {
			t.Errorf(
				"%s: %v alocações por chamada, esperado %v",
				c.nome,
				got,
				c.quer,
			)
		}
	}
}
