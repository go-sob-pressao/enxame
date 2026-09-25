// Package trava — o teste com synctest que nunca termina.
package trava

import (
	"sync"
	"time"
)

// Cache guarda um valor e o renova periodicamente.
type Cache struct {
	mu    sync.Mutex
	valor string
}

// livro:inicio renovar-defeito

// RenovarLento segura a trava durante toda a renovação, que inclui
// esperar o serviço remoto — aqui, um Sleep de 1 s.
func (c *Cache) RenovarLento(buscar func() string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	time.Sleep(time.Second) // a chamada remota
	c.valor = buscar()
}

// livro:fim renovar-defeito

// livro:inicio renovar-correto

// Renovar espera o serviço remoto sem a trava, e só a segura para
// trocar o valor.
func (c *Cache) Renovar(buscar func() string) {
	time.Sleep(time.Second) // a chamada remota
	v := buscar()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.valor = v
}

// livro:fim renovar-correto

// Valor devolve o valor atual.
func (c *Cache) Valor() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valor
}
