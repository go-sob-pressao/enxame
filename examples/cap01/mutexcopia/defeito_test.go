//go:build defeito

package main

import "testing"

// Sem -race, o total costuma sair menor que o esperado: incrementos
// perdidos. Com -race, o detector aponta as duas pilhas e o teste
// falha.
func TestCopiaNaoProtege(t *testing.T) {
	t.Logf("total: %d (esperado 100)", somar(100))
}
