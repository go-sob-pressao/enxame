//go:build defeito

package errignorado

// livro:inicio err-ignorado-defeito

// RegistrarPedido "sempre funciona": o erro do Commit é jogado fora com
// _. O compilador aceita, o errcheck padrão aceita (o _ é explícito), e
// o cliente recebe "pedido registrado" para um pedido que não existe.
func RegistrarPedido(tx *Tx, pedido string) string {
	tx.Inserir(pedido)
	_ = tx.Commit()
	return "pedido registrado"
}

// livro:fim err-ignorado-defeito
