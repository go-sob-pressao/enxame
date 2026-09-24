//go:build defeito

package fechamento

import (
	"testing"
	"time"
)

// O pânico acontece numa goroutine de produtor, fora do alcance do
// recover do teste: o processo inteiro cai com "send on closed
// channel". É o comportamento do enigma — por isso este teste só roda
// quando pedido.
func TestUmProdutorFechaEnquantoOutroEnvia(t *testing.T) {
	if testing.Short() {
		t.Skip("derruba o processo de propósito; rode sem -short")
	}
	for range Produzir([][]int{{1}, make([]int, 100_000)}) {
	}
	// O range acaba no primeiro close; o pânico vem da goroutine do
	// outro produtor. Sem esta espera, o teste pode terminar antes dele.
	time.Sleep(time.Second)
}
