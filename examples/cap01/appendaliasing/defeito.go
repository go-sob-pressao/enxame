//go:build defeito

package main

// livro:inicio append-defeito

// derivar cria duas variações de base. Com capacidade sobrando, os dois
// append escrevem no MESMO array: o segundo sobrescreve o primeiro.
func derivar(itens []int) (a, b []int) {
	base := make([]int, len(itens), 10)
	copy(base, itens)
	a = append(base, 100)
	b = append(base, 200)
	return a, b
}

// livro:fim append-defeito
