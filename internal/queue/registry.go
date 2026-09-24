package queue

import "time"

// QueueConfig é a configuração de uma fila, recarregável sem reiniciar.
type QueueConfig struct {
	Concurrency int
	PollTimeout time.Duration
}

// livro:inicio registry-corrida

// Registry guarda a configuração corrente de cada fila. O
// recarregamento de configuração chama Set; cada pool chama Get a cada
// ciclo de busca.
type Registry struct {
	filas map[string]QueueConfig
}

// NewRegistry cria um registro vazio.
func NewRegistry() *Registry {
	return &Registry{filas: map[string]QueueConfig{}}
}

// Set grava a configuração de uma fila.
func (r *Registry) Set(nome string, c QueueConfig) { r.filas[nome] = c }

// Get devolve a configuração de uma fila.
func (r *Registry) Get(nome string) (QueueConfig, bool) {
	c, ok := r.filas[nome]
	return c, ok
}

// livro:fim registry-corrida
