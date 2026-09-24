// Command sombra: o err sombreado por := num escopo interno.
package main

import (
	"errors"
	"fmt"
)

// ErrIndisponivel simula a falha de uma consulta.
var ErrIndisponivel = errors.New("estoque indisponível")

// consultar falha para o item 3.
func consultar(id int) (int, error) {
	if id == 3 {
		return 0, ErrIndisponivel
	}
	return id * 10, nil
}

func main() {
	total, err := somarEstoque([]int{1, 2, 3, 4})
	fmt.Printf("total=%d err=%v\n", total, err)
}
