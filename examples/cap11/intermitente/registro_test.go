//go:build !defeito

package intermitente

import (
	"testing"
	"testing/synctest"
)

// livro:inicio sem-aposta

// Dentro da bolha, synctest.Wait devolve só quando todas as outras
// goroutines da bolha terminaram ou estão bloqueadas de vez: a gravação
// já aconteceu, sempre.
func TestNotificarGrava(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var r Registro
		r.Notificar("login")
		synctest.Wait()
		if r.Total() != 1 {
			t.Fatalf("total %d, esperado 1", r.Total())
		}
	})
}

// livro:fim sem-aposta
