package heartbeat

import (
	"context"
	"errors"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// ErrPerdida indica que a tentativa não pertence mais a este worker: o
// batimento foi recusado porque o job foi resgatado e entregue a outro.
var ErrPerdida = errors.New("tentativa perdida: o job foi resgatado")

// livro:inicio heartbeat

// Run bate a cada intervalo, enquanto ctx estiver vivo. Um batimento
// recusado pelo domínio quer dizer que a tentativa já não é deste
// worker — foi resgatada enquanto ele trabalhava, talvez do outro lado
// de uma rede partida —, e Run chama perdeu e para: continuar seria
// executar em dobro. Qualquer outro erro é tratado como passageiro; o
// próximo batimento tenta de novo, e o prazo do resgate decide.
func Run(
	ctx context.Context,
	intervalo time.Duration,
	bater func(context.Context) error,
	perdeu func(error),
) {
	t := time.NewTicker(intervalo)
	defer t.Stop()
	for {
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
		err := bater(ctx)
		if errors.Is(err, job.ErrInvalidTransition) {
			perdeu(errors.Join(ErrPerdida, err))
			return
		}
	}
}

// livro:fim heartbeat
