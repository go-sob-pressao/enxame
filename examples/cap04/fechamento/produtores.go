// Package fechamento — vários produtores escrevendo num canal. Quem
// fecha?
package fechamento

// livro:inicio produtores
// Produzir inicia um produtor por fonte e devolve um único canal com
// tudo.
func Produzir(fontes [][]int) <-chan int {
	saida := make(chan int)
	produzir(fontes, saida)
	return saida
}

// livro:fim produtores
