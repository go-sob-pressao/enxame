package contador

import (
	"sync"
	"testing"
)

type somador interface {
	Somar(int)
	Total() int
}

func martelar(s somador, goroutines, vezes int) {
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range vezes {
				s.Somar(1)
			}
		})
	}
	wg.Wait()
}

func TestOsDoisContamCerto(t *testing.T) {
	canal := NovoComCanal()
	defer canal.Parar()
	for _, s := range []somador{canal, &ComMutex{}} {
		martelar(s, 8, 1000)
		if s.Total() != 8000 {
			t.Errorf("%T: %d", s, s.Total())
		}
	}
}

func BenchmarkComCanal(b *testing.B) {
	c := NovoComCanal()
	defer c.Parar()
	for b.Loop() {
		c.Somar(1)
	}
}

func BenchmarkComMutex(b *testing.B) {
	c := &ComMutex{}
	for b.Loop() {
		c.Somar(1)
	}
}
