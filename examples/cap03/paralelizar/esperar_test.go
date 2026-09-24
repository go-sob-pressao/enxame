package paralelizar

import (
	"fmt"
	"testing"
	"time"
)

// Mil tarefas de 1 ms de espera, de 1 a mil goroutines.
func BenchmarkEsperar(b *testing.B) {
	for _, partes := range []int{1, 8, 64, 1000} {
		b.Run(fmt.Sprintf("partes=%d", partes), func(b *testing.B) {
			for b.Loop() {
				Esperar(1000, partes, time.Millisecond)
			}
		})
	}
}
