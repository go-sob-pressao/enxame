//go:build !defeito

package errignorado

import "fmt"

// livro:inicio err-ignorado-correto

// RegistrarPedido devolve o erro do Commit a quem precisa decidir.
func RegistrarPedido(tx *Tx, pedido string) (string, error) {
	tx.Inserir(pedido)
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("registrar pedido %s: %w", pedido, err)
	}
	return "pedido registrado", nil
}

// livro:fim err-ignorado-correto
