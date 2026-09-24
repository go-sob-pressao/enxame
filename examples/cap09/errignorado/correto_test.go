//go:build !defeito

package errignorado

import (
	"errors"
	"testing"
)

func TestFalhaDoCommitChegaAoChamador(t *testing.T) {
	var gravado []string
	_, err := RegistrarPedido(
		&Tx{falharCommit: true, gravado: &gravado},
		"pedido-42",
	)
	if !errors.Is(err, ErrConflito) || len(gravado) != 0 {
		t.Fatalf("err=%v gravado=%v", err, gravado)
	}
}
