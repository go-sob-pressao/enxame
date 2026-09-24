package queue_test

import (
	"sync"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/queue"
)

// Recarregar a configuração enquanto os pools leem: o cenário de
// produção.
func TestRegistryRecarregaEnquantoLeem(t *testing.T) {
	r := queue.NewRegistry()
	r.Set(
		"pagamentos",
		queue.Config{Concurrency: 10, PollTimeout: time.Second},
	)
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 100 {
			r.Set(
				"pagamentos",
				queue.Config{
					Concurrency: 10 + i,
					PollTimeout: time.Second,
				},
			)
		}
	})
	for range 4 {
		wg.Go(func() {
			for range 100 {
				if c, ok := r.Get("pagamentos"); !ok ||
					c.Concurrency < 10 {
					t.Errorf("config inválida: %+v", c)
				}
			}
		})
	}
	wg.Wait()
}
