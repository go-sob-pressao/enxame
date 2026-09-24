//go:build !defeito

package main

import "sync"

// livro:inicio mutex-copia-correto
// Contador é usado sempre por ponteiro: existe um único Mutex.
type Contador struct {
	mu     sync.Mutex
	totais []int
}

func incrementar(c *Contador) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totais[0]++
}

// livro:fim mutex-copia-correto

func somar(n int) int {
	c := &Contador{totais: make([]int, 1)}
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() { incrementar(c) })
	}
	wg.Wait()
	return c.totais[0]
}
