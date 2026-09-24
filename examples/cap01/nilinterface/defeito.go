//go:build defeito

package main

// livro:inicio nil-interface-defeito

// validar devolve um ponteiro nil dentro de uma interface. A interface
// guarda o par (tipo, valor) = (*ErroValidacao, nil), e interface com
// tipo não é nil.
func validar(nome string) error {
	var err *ErroValidacao
	if nome == "" {
		err = &ErroValidacao{Campo: "nome"}
	}
	return err
}

// livro:fim nil-interface-defeito
