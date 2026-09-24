//go:build !defeito

package main

// livro:inicio sombra-correto

func somarEstoque(ids []int) (int, error) {
	total := 0
	for _, id := range ids {
		qtd, err := consultar(id)
		if err != nil {
			return total, err
		}
		total += qtd
	}
	return total, nil
}

// livro:fim sombra-correto
