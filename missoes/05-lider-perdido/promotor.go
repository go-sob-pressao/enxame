package liderperdido

import (
	"context"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

// livro:inicio missao-05

// Promotor torna disponíveis os jobs agendados das partições do nó.
// Para não varrer a tabela inteira a cada passada, cada partição tem
// uma marca: a passada só olha os jobs cuja hora caiu entre a marca e
// agora, e a marca avança.
type Promotor struct {
	Store  *postgres.Store
	Posses *partition.Posses

	marca map[int]time.Time
}

// Passar promove, em cada partição do nó, o que venceu desde a marca.
func (p *Promotor) Passar(ctx context.Context, agora time.Time) error {
	if p.marca == nil {
		p.marca = map[int]time.Time{}
	}
	for part, token := range p.Posses.Tokens() {
		desde, ok := p.marca[part]
		if !ok {
			desde = agora // partição nova: começa a contar daqui
		}
		s := p.Store.ComCerca(postgres.Cerca{Particao: part,
			RangeID: token})
		ids, err := s.Vencidos(ctx, part, desde, agora)
		if err != nil {
			return err
		}
		for _, jid := range ids {
			if err := promover(ctx, s, jid, agora); err != nil {
				return err
			}
		}
		p.marca[part] = agora
	}
	return nil
}

// livro:fim missao-05

func promover(ctx context.Context, s *postgres.Store, jid id.JobID,
	agora time.Time) error {
	_, err := s.Decide(ctx, jid, func(j job.Job) ([]job.Event, error) {
		return job.MakeAvailable(j, agora)
	})
	return err
}
