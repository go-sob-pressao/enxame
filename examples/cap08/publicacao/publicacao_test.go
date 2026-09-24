package publicacao

import (
	"sync"
	"testing"
)

func TestLeiturasVeemConfigsCompletas(t *testing.T) {
	var a Atual
	a.Publicar(&Config{Concorrencia: 1, Filas: []string{"a"}})
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			for range 1000 {
				if i == 0 {
					a.Publicar(
						&Config{
							Concorrencia: 2,
							Filas:        []string{"a", "b"},
						},
					)
					continue
				}
				c := a.Ler()
				if c.Concorrencia != len(c.Filas) {
					t.Errorf("config pela metade: %+v", c)
					return
				}
			}
		})
	}
	wg.Wait()
}
