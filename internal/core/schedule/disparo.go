package schedule

import (
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// livro:inicio disparo

// Disparo decide o job da janela vencida de um agendamento e a próxima
// janela. A chave única do job é a janela prevista — não a hora em que
// o disparo aconteceu —, então disparar a mesma janela duas vezes
// produz a mesma chave, e o banco guarda um job só. Janelas perdidas
// enquanto ninguém disparava são puladas: a próxima é a primeira
// depois de agora. O id do job fica em branco: quem enfileira o gera,
// na borda.
func Disparo(
	sc Schedule,
	now time.Time,
) (job.Spec, time.Time, error) {
	e, err := Parse(sc.Expr)
	if err != nil {
		return job.Spec{}, time.Time{}, err
	}
	loc, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		return job.Spec{}, time.Time{}, err
	}
	spec := job.Spec{
		Namespace: sc.Namespace, Queue: sc.Queue, Kind: sc.Kind,
		Args:      sc.Args,
		UniqueKey: id.WindowKey(sc.Namespace, sc.ID, sc.NextFire),
	}
	return spec, e.Next(later(sc.NextFire, now), loc), nil
}

// livro:fim disparo

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
