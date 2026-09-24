// Package publicacao — trocar a configuração enquanto outras goroutines
// a leem.
package publicacao

import "sync/atomic"

// Config é imutável depois de publicada.
type Config struct {
	Concorrencia int
	Filas        []string
}

// livro:inicio publicacao

// Atual guarda a configuração vigente. Store publica uma Config nova
// inteira; Load devolve sempre uma Config completa — nunca meio velha,
// meio nova.
type Atual struct{ p atomic.Pointer[Config] }

// Publicar troca a configuração.
func (a *Atual) Publicar(c *Config) { a.p.Store(c) }

// Ler devolve a configuração vigente. Quem lê não modifica.
func (a *Atual) Ler() *Config { return a.p.Load() }

// livro:fim publicacao
