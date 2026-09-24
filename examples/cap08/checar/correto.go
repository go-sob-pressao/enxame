//go:build !defeito

package checar

import "sync"

// livro:inicio checar-correto

// Cache garante uma abertura por endpoint: o sync.OnceValue de cada
// endpoint é criado uma vez (LoadOrStore) e executa abrir uma vez.
type Cache struct{ m sync.Map }

// Obter devolve a conexão do endpoint, abrindo na primeira vez.
func (c *Cache) Obter(endpoint string) *Conexao {
	v, _ := c.m.LoadOrStore(
		endpoint,
		sync.OnceValue(func() *Conexao { return abrir(endpoint) }),
	)
	return v.(func() *Conexao)()
}

// livro:fim checar-correto
