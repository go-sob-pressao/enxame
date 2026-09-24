// Package flag — a flag de parada que o compilador leu uma vez só.
package flag

// Trabalhador incrementa um contador até receber o pedido de parada.
type Trabalhador interface {
	Parar()
	Trabalhar(n *int)
}
