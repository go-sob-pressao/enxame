//go:build defeito

package flag

// livro:inicio flag-defeito

// ComBool usa um bool comum. Sem sincronização, não há happens-before
// entre a escrita em Parar e a leitura no laço: o compilador pode ler
// parar uma vez, antes do laço, e girar para sempre.
type ComBool struct{ parar bool }

// Parar pede que o trabalho termine.
func (c *ComBool) Parar() { c.parar = true }

// Trabalhar gira até parar.
func (c *ComBool) Trabalhar(n *int) {
	for !c.parar {
		*n++
	}
}

// livro:fim flag-defeito

// Novo devolve a versão com defeito.
func Novo() Trabalhador { return &ComBool{} }
