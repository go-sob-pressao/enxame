// Package buscaativa — select com default num laço: a espera que queima
// CPU.
package buscaativa

// livro:inicio busca-ativa

// EsperarComDefault espera um valor girando: o default torna o select
// não bloqueante, e o laço o repete milhões de vezes por segundo.
// Devolve quantas voltas deu — cada uma é CPU gasta sem trabalho.
func EsperarComDefault(c <-chan int) (valor, voltas int) {
	for {
		select {
		case v := <-c:
			return v, voltas
		default:
			voltas++
		}
	}
}

// EsperarBloqueando espera sem gastar CPU: a goroutine dorme até o
// valor chegar.
func EsperarBloqueando(c <-chan int) int {
	return <-c
}

// livro:fim busca-ativa
