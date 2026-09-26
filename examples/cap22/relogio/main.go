// Command relogio mostra os dois relógios de um time.Time e o que a
// serialização faz com eles (Capítulo 22).
//
//	go run ./examples/cap22/relogio
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// livro:inicio relogio

func main() {
	inicio := time.Now() // parede + leitura monotônica
	fmt.Println("inicio:        ", inicio)

	b, _ := json.Marshal(inicio) // o banco, a rede, outro nó
	var copia time.Time
	_ = json.Unmarshal(b, &copia)
	fmt.Println("depois do JSON:", copia)
	//nolint:staticcheck // o == é justamente o que o exemplo mostra
	fmt.Println("iguais? ==", inicio == copia,
		"· Equal", inicio.Equal(copia))

	time.Sleep(50 * time.Millisecond)
	agora := time.Now()
	// Sub usa o monotônico quando os dois lados o têm; senão, a parede.
	fmt.Println("agora − inicio:", agora.Sub(inicio), "(monotônico)")
	fmt.Println("agora − copia: ", agora.Sub(copia), "(parede)")
	fmt.Println("Round(0):      ", inicio.Round(0))
}

// livro:fim relogio
