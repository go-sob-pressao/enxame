//go:build defeito

package intermitente

import (
	"testing"
	"time"
)

// livro:inicio aposta

// O teste aposta que um instante basta para a goroutine de Notificar
// rodar. Quase sempre basta. (No código real a aposta era de 1 ms, e
// falhava sob a carga da CI; aqui ela é de 5 µs para que a falha
// apareça também numa máquina ociosa.)
func TestNotificarGrava(t *testing.T) {
	var r Registro
	r.Notificar("login")
	time.Sleep(5 * time.Microsecond) //nolint:forbidigo // o defeito
	if r.Total() != 1 {
		t.Fatalf("total %d, esperado 1", r.Total())
	}
}

// livro:fim aposta
