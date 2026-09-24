package dispatcher_test

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/engine/dispatcher"
)

// Mesma chave: exclusão mútua garantida pelos dois.
func TestMesmaChaveSerializa(t *testing.T) {
	for _, kl := range []dispatcher.KeyLock{&dispatcher.SingleMutex{}, dispatcher.NewStriped(64)} {
		total := 0
		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				unlock := kl.Lock("pedido-42")
				defer unlock()
				total++
			})
		}
		wg.Wait()
		if total != 100 {
			t.Fatalf("%T: total %d", kl, total)
		}
	}
}

// livro:inicio bench-striped

// Cada operação segura a trava por 20 µs, como uma escrita no
// histórico. 64 goroutines, cada uma com a sua chave.
func BenchmarkTravaPorChave(b *testing.B) {
	casos := map[string]dispatcher.KeyLock{
		"mutex-por-particao": &dispatcher.SingleMutex{},
		"striped-64":         dispatcher.NewStriped(64),
	}
	for _, nome := range []string{"mutex-por-particao", "striped-64"} {
		kl := casos[nome]
		b.Run(nome, func(b *testing.B) {
			b.SetParallelism(64)
			var n int64
			var mu sync.Mutex
			b.RunParallel(func(pb *testing.PB) {
				mu.Lock()
				n++
				chave := "job-" + strconv.FormatInt(n, 10)
				mu.Unlock()
				for pb.Next() {
					unlock := kl.Lock(chave)
					trabalho(20 * time.Microsecond)
					unlock()
				}
			})
		})
	}
}

// livro:fim bench-striped

// trabalho ocupa a CPU por d sem dormir (dormir liberaria a trava do
// teste).
func trabalho(d time.Duration) {
	fim := time.Now().Add(d)
	for time.Now().Before(fim) {
		_ = fmt.Sprint()
	}
}
