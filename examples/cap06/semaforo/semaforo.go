// Package semaforo — limitar concorrência com channel e com
// x/sync/semaphore.
package semaforo

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// livro:inicio semaforo-canal

// ComCanal executa tarefas com no máximo limite ao mesmo tempo: o
// buffer do canal é o número de vagas.
func ComCanal(tarefas []func(), limite int) {
	vagas := make(chan struct{}, limite)
	var wg sync.WaitGroup
	for _, tarefa := range tarefas {
		vagas <- struct{}{} // ocupa uma vaga; bloqueia se não houver
		wg.Go(func() {
			defer func() { <-vagas }()
			tarefa()
		})
	}
	wg.Wait()
}

// livro:fim semaforo-canal

// livro:inicio semaforo-xsync

// ComSemaforo faz o mesmo com peso por tarefa e espera cancelável.
func ComSemaforo(
	ctx context.Context,
	tarefas []func(),
	limite int64,
) error {
	sem := semaphore.NewWeighted(limite)
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, tarefa := range tarefas {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		wg.Go(func() {
			defer sem.Release(1)
			tarefa()
		})
	}
	return nil
}

// livro:fim semaforo-xsync
