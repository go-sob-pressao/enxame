// Command archcheck valida as regras de dependência entre camadas do
// Enxame.
//
// Regra fundamental: internal/core não importa nada de internal/, não
// usa bibliotecas de terceiros, não faz I/O e não lê o relógio. O
// domínio é puro. Se qualquer regra quebrar, o build falha.
//
// As regras estão em regras.go; o Capítulo 2 as escreve, e cada
// capítulo que cria uma camada nova acrescenta a sua.
package main

import (
	"fmt"
	"os"
)

func main() {
	vs, err := verificar(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(2)
	}
	for _, v := range vs {
		fmt.Println("violação:", v)
	}
	if len(vs) > 0 {
		fmt.Printf("\n%d violação(ões) de arquitetura\n", len(vs))
		os.Exit(1)
	}
	fmt.Println("arquitetura ok")
}
