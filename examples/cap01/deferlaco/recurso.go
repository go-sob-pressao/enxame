package main

import "sync/atomic"

// Recurso simula um arquivo: conta quantos estão abertos ao mesmo
// tempo.
type Recurso struct{ pool *Pool }

// Pool registra abertos agora e o pico.
type Pool struct {
	abertos atomic.Int64
	pico    atomic.Int64
}

// Abrir abre um recurso.
func (p *Pool) Abrir() *Recurso {
	n := p.abertos.Add(1)
	for {
		atual := p.pico.Load()
		if n <= atual || p.pico.CompareAndSwap(atual, n) {
			break
		}
	}
	return &Recurso{pool: p}
}

// Close fecha o recurso.
func (r *Recurso) Close() error {
	r.pool.abertos.Add(-1)
	return nil
}
