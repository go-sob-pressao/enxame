package paralelizar

import (
	"sync"
	"time"
)

// livro:inicio esperar-em-partes

// Esperar simula trabalho de I/O: tarefas que passam o tempo esperando
// uma resposta, não calculando. Divide as tarefas entre partes
// goroutines, e cada uma espera pelas suas, uma de cada vez.
func Esperar(tarefas, partes int, espera time.Duration) {
	partes = max(1, min(partes, tarefas))
	var wg sync.WaitGroup
	for p := range partes {
		wg.Go(func() {
			for t := p; t < tarefas; t += partes {
				time.Sleep(espera)
			}
		})
	}
	wg.Wait()
}

// livro:fim esperar-em-partes
