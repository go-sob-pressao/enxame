// Package paralelizar mostra quando mais goroutines deixam o trabalho
// mais lento.
package paralelizar

import "sync"

// livro:inicio somar-em-partes

// SomarQuadrados divide os dados em partes e soma cada parte numa
// goroutine. Com partes demais, o custo de criar e coordenar goroutines
// passa o ganho.
func SomarQuadrados(dados []int64, partes int) int64 {
	if partes < 1 {
		partes = 1
	}
	tamanho := (len(dados) + partes - 1) / partes
	parciais := make([]int64, partes)
	var wg sync.WaitGroup
	for p := range partes {
		inicio := min(p*tamanho, len(dados))
		fim := min(inicio+tamanho, len(dados))
		wg.Go(func() {
			var s int64
			for _, v := range dados[inicio:fim] {
				s += v * v
			}
			parciais[p] = s
		})
	}
	wg.Wait()
	var total int64
	for _, s := range parciais {
		total += s
	}
	return total
}

// livro:fim somar-em-partes
