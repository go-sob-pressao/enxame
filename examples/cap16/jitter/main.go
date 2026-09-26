// Command jitter simula mil clientes que falharam no mesmo instante e
// tentam de novo seguindo a política de retry do Enxame, com e sem
// jitter. Imprime, a cada 250 ms, quantas tentativas chegam ao servidor
// — os dados da Figura 16.2 — e o pico em 10 ms.
//
//	go run ./examples/cap16/jitter > jitter.csv
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/policy"
)

func main() {
	const clientes, tentativas = 1000, 5
	r := policy.Retry{Base: time.Second, Max: time.Minute}
	sorte := rand.New(rand.NewPCG(16, 2)) // semente fixa: reproduzível
	const faixa = 250 * time.Millisecond
	sem, com := map[int]int{}, map[int]int{}
	picoSem, picoCom := map[int]int{}, map[int]int{} // por 10 ms
	for range clientes {
		var t1, t2 time.Duration
		for a := 1; a <= tentativas; a++ {
			t1 += r.Teto(a)
			t2 += r.Delay(a, sorte.Float64)
			sem[int(t1/faixa)]++
			com[int(t2/faixa)]++
			picoSem[int(t1/(10*time.Millisecond))]++
			picoCom[int(t2/(10*time.Millisecond))]++
		}
	}
	fmt.Println("inicio_s,sem_jitter,com_jitter")
	for f := range 32 * 4 {
		fmt.Printf("%.2f,%d,%d\n", float64(f)/4, sem[f], com[f])
	}
	fmt.Fprintf(os.Stderr,
		"pico em 10 ms: sem jitter %d, com jitter %d\n",
		maximo(picoSem), maximo(picoCom))
}

func maximo(m map[int]int) int {
	x := 0
	for _, v := range m {
		x = max(x, v)
	}
	return x
}
