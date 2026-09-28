package http

import "slices"

// livro:inicio missao-06-gabarito

// kindPermitido diz se o kind está na lista de kinds que a API aceita.
// A lista vem da configuração (-kinds); vazia, aceita qualquer um. Uma
// busca na lista, sem expressão regular: quatrocentas comparações de
// string custam menos que montar e compilar uma expressão a cada
// requisição, e não alocam nada.
func kindPermitido(kinds []string, kind string) bool {
	return len(kinds) == 0 || slices.Contains(kinds, kind)
}

// livro:fim missao-06-gabarito
