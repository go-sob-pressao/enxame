package avalanche

import (
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/policy"
)

// livro:inicio missao-04-gabarito

// ProximaTentativa diz quando repetir a entrega que falhou na tentativa
// n (1, 2, 3…), a partir de agora: backoff exponencial até 10 minutos,
// com jitter total — cada entrega sorteia a sua espera entre zero e o
// teto, e as que falharam juntas deixam de voltar juntas.
func ProximaTentativa(
	n int,
	agora time.Time,
	sorte func() float64,
) time.Time {
	r := policy.Retry{Base: time.Second, Max: 10 * time.Minute}
	return agora.Add(r.Delay(n, sorte))
}

// livro:fim missao-04-gabarito
