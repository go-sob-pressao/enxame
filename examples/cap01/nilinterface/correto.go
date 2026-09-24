//go:build !defeito

package main

// livro:inicio nil-interface-correto

func validar(nome string) error {
	if nome == "" {
		return &ErroValidacao{Campo: "nome"}
	}
	return nil
}

// livro:fim nil-interface-correto
