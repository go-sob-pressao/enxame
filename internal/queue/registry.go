package queue

import (
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

// Config é a configuração de uma fila, recarregável sem reiniciar.
type Config struct {
	Concurrency int
	PollTimeout time.Duration
}

// livro:inicio registry-corrigido

// Registry guarda a configuração corrente de cada fila. Leituras são
// muito mais frequentes que recarregamentos, então o mapa é imutável
// depois de publicado: Set copia, altera a cópia e publica o novo mapa
// inteiro com atomic.Pointer; Get lê sem trava nenhuma.
type Registry struct {
	// serializa os Set: dois recarregamentos não se perdem
	escrita sync.Mutex
	filas   atomic.Pointer[map[string]Config]
}

// NewRegistry cria um registro vazio.
func NewRegistry() *Registry {
	r := &Registry{}
	r.filas.Store(&map[string]Config{})
	return r
}

// Set grava a configuração de uma fila.
func (r *Registry) Set(nome string, c Config) {
	r.escrita.Lock()
	defer r.escrita.Unlock()
	novo := maps.Clone(*r.filas.Load())
	novo[nome] = c
	r.filas.Store(&novo)
}

// Get devolve a configuração de uma fila.
func (r *Registry) Get(nome string) (Config, bool) {
	c, ok := (*r.filas.Load())[nome]
	return c, ok
}

// livro:fim registry-corrigido
