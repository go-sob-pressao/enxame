//go:build defeito

package main

import "sync"

// livro:inicio mutex-copia-defeito
// Contador parece protegido: tem um Mutex e trava antes de escrever.
type Contador struct {
	mu     sync.Mutex
	totais []int
}

// incrementar recebe o Contador por valor. Cada chamada trava a própria
// cópia do Mutex; o slice copiado aponta para o MESMO array de todas.
func incrementar(c Contador) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totais[0]++
}

// livro:fim mutex-copia-defeito

func somar(n int) int {
	c := Contador{totais: make([]int, 1)}
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() { incrementar(c) })
	}
	wg.Wait()
	return c.totais[0]
}
