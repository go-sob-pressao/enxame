package engine

import (
	"context"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// livro:inicio store-consumidor

// Store é o que o motor precisa do armazenamento, declarado aqui, por
// quem o usa. As implementações — memória, SQLite e Postgres — não
// importam este pacote: satisfazem a interface porque têm os métodos.
type Store interface {
	// Insert grava o job e o primeiro histórico, atomicamente.
	Insert(ctx context.Context, j job.Job, evs []job.Event) error
	// Get devolve a projeção e a versão, para o lock otimista.
	Get(ctx context.Context, jid id.JobID) (job.Job, int64, error)
	// History devolve os eventos do job, em ordem.
	History(ctx context.Context, jid id.JobID) ([]job.Event, error)
	// Update grava a nova projeção e acrescenta eventos, se a versão
	// gravada ainda for version.
	Update(
		ctx context.Context,
		j job.Job,
		evs []job.Event,
		version int64,
	) error
	// Next devolve o próximo job disponível da fila, sem reservá-lo.
	Next(
		ctx context.Context,
		queue string,
	) (job.Job, int64, bool, error)
}

// livro:fim store-consumidor
