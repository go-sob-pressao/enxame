// Command nilinterface: o erro que não é nil dentro de uma interface.
package main

import "fmt"

// ErroValidacao descreve um campo inválido.
type ErroValidacao struct{ Campo string }

func (e *ErroValidacao) Error() string {
	return "campo inválido: " + e.Campo
}

func main() {
	if err := validar("Ana"); err != nil {
		fmt.Println("falhou:", err)
		return
	}
	fmt.Println("nome válido")
}
