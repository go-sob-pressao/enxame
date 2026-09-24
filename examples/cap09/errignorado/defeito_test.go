//go:build defeito

package errignorado

import "testing"

func TestPedidoPerdidoEmSilencio(t *testing.T) {
	var gravado []string
	msg := RegistrarPedido(
		&Tx{falharCommit: true, gravado: &gravado},
		"pedido-42",
	)
	t.Logf(
		"resposta ao cliente: %q; linhas gravadas: %d",
		msg,
		len(gravado),
	)
}
