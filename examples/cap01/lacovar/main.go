// Command lacovar: a captura de variável de laço que morreu no Go 1.22.
//
// Este módulo declara go 1.21 no go.mod, então o laço tem a semântica
// antiga mesmo compilado pelo Go 1.27. Troque para go 1.22 e rode de
// novo.
package main

import "fmt"

func main() {
	var funcs []func() int
	for i := 0; i < 3; i++ {
		funcs = append(funcs, func() int { return i })
	}
	for _, f := range funcs {
		fmt.Print(f(), " ")
	}
	fmt.Println()
}
