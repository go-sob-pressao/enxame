//go:build !defeito

package fechamento

import "sync"

// livro:inicio fechamento-correto

// produzir: os produtores só enviam. Uma goroutine a mais, dona do
// fechamento, espera todos terminarem e fecha uma única vez.
func produzir(fontes [][]int, saida chan<- int) {
	var wg sync.WaitGroup
	for _, fonte := range fontes {
		wg.Go(func() {
			for _, v := range fonte {
				saida <- v
			}
		})
	}
	go func() {
		wg.Wait()
		close(saida)
	}()
}

// livro:fim fechamento-correto
