//go:build defeito

package checar

import "sync"

// livro:inicio checar-defeito

// Cache usa sync.Map, que é seguro para uso concorrente. Cada operação
// é atômica — mas "verificar e depois agir" são DUAS operações: entre o
// Load e o Store, outra goroutine faz o mesmo Load e também abre uma
// conexão.
type Cache struct{ m sync.Map }

// Obter devolve a conexão do endpoint, abrindo se não existir.
func (c *Cache) Obter(endpoint string) *Conexao {
	if v, ok := c.m.Load(endpoint); ok {
		return v.(*Conexao)
	}
	conn := abrir(endpoint)
	c.m.Store(endpoint, conn)
	return conn
}

// livro:fim checar-defeito
