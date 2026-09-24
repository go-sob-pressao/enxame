//go:build defeito

package fechamento

// livro:inicio fechamento-defeito

// produzir: cada produtor fecha o canal ao terminar a sua fonte. O
// primeiro que terminar fecha; o seguinte que tentar enviar entra em
// pânico.
func produzir(fontes [][]int, saida chan<- int) {
	for _, fonte := range fontes {
		go func() {
			defer close(saida)
			for _, v := range fonte {
				saida <- v
			}
		}()
	}
}

// livro:fim fechamento-defeito
