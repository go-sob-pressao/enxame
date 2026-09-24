// Package contador — o mesmo contador com channel e com Mutex.
package contador

import "sync"

// livro:inicio contador-canal

// ComCanal protege o total com uma goroutine dona e um canal de
// pedidos. Funciona — e é o Anti-Pattern #1: não há propriedade a
// transferir, só um número a proteger.
type ComCanal struct {
	incrementos chan int
	leituras    chan chan int
}

// NovoComCanal inicia a goroutine dona. Pare com Parar.
func NovoComCanal() *ComCanal {
	c := &ComCanal{
		incrementos: make(chan int),
		leituras:    make(chan chan int),
	}
	go func() {
		total := 0
		for {
			select {
			case d, ok := <-c.incrementos:
				if !ok {
					return
				}
				total += d
			case resp := <-c.leituras:
				resp <- total
			}
		}
	}()
	return c
}

// Somar adiciona d.
func (c *ComCanal) Somar(d int) { c.incrementos <- d }

// Total devolve o total.
func (c *ComCanal) Total() int {
	resp := make(chan int)
	c.leituras <- resp
	return <-resp
}

// Parar encerra a goroutine dona.
func (c *ComCanal) Parar() { close(c.incrementos) }

// livro:fim contador-canal

// livro:inicio contador-mutex

// ComMutex faz o mesmo em três linhas por método.
type ComMutex struct {
	mu    sync.Mutex
	total int
}

// Somar adiciona d.
func (c *ComMutex) Somar(d int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total += d
}

// Total devolve o total.
func (c *ComMutex) Total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

// livro:fim contador-mutex
