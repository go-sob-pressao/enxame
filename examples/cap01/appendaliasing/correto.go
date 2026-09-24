//go:build !defeito

package main

import "slices"

// livro:inicio append-correto

// slices.Clip corta a capacidade ao tamanho: o append seguinte é
// obrigado a alocar um array novo para cada variação.
func derivar(itens []int) (a, b []int) {
	base := make([]int, len(itens), 10)
	copy(base, itens)
	base = slices.Clip(base)
	a = append(base, 100)
	b = append(base, 200)
	return a, b
}

// livro:fim append-correto
