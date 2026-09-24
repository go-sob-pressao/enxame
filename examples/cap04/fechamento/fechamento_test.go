//go:build !defeito

package fechamento

import "testing"

func TestTodosOsValoresUmFechamento(t *testing.T) {
	soma := 0
	for v := range Produzir([][]int{{1, 2, 3}, {10, 20}, {100}}) {
		soma += v
	}
	if soma != 136 {
		t.Fatalf("soma %d, esperado 136", soma)
	}
}
