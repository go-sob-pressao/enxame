//go:build defeito

package main

// livro:inicio sombra-defeito

func somarEstoque(ids []int) (total int, err error) {
	for _, id := range ids {
		qtd, err := consultar(id) // := cria um NOVO err, só deste bloco
		if err != nil {
			break
		}
		total += qtd
	}
	return total, err // o err de fora nunca foi atribuído: sempre nil
}

// livro:fim sombra-defeito
