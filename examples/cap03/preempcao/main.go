// Command preempcao: o laço apertado que travava o mundo antes do Go
// 1.14.
//
//	go run ./examples/cap03/preempcao                          termina
//	GODEBUG=asyncpreemptoff=1 go run ./examples/cap03/preempcao
//	    trava (Ctrl-C)
package main

import (
	"fmt"
	"runtime"
	"time"
)

// livro:inicio laco-apertado

// girar não chama função nenhuma: sem preempção assíncrona, o
// escalonador não tem onde interromper esta goroutine.
func girar(n *uint64) {
	for {
		*n++
	}
}

func main() {
	// um P só: quem gira ocupa o único processador
	runtime.GOMAXPROCS(1)
	var n uint64
	go girar(&n)
	// a goroutine main precisa voltar a rodar
	time.Sleep(100 * time.Millisecond)
	fmt.Println(
		"main voltou a rodar:",
		"a preempção assíncrona interrompeu o laço",
	)
}

// livro:fim laco-apertado
