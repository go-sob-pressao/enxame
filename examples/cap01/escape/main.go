// Command escape: o que parece stack e está na heap.
//
//	go build -gcflags=-m ./examples/cap01/escape
package main

import "fmt"

// livro:inicio escape

// Ponto é pequeno e sem ponteiros.
type Ponto struct{ X, Y int }

// naPilha devolve o valor: a cópia sai, nada escapa.
func naPilha() Ponto {
	p := Ponto{1, 2}
	return p
}

// naHeap devolve o endereço de uma variável local: ela precisa
// sobreviver à função, então o compilador a move para a heap.
func naHeap(x, y int) *Ponto {
	p := Ponto{x, y}
	return &p
}

// viaInterface converte para interface: o valor é copiado para a heap
// porque a interface guarda um ponteiro para ele.
func viaInterface(x, y int) any {
	p := Ponto{x, y}
	return p
}

// livro:fim escape

func main() {
	fmt.Println(naPilha(), *naHeap(3, 4), viaInterface(5, 6))
}
