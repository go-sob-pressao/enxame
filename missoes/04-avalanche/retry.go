package avalanche

import "time"

// livro:inicio missao-04

// ProximaTentativa diz quando repetir a entrega que falhou na tentativa
// n (1, 2, 3…), a partir de agora: backoff exponencial, dobrando a cada
// falha, até 17 minutos. sorte devolve um número em [0, 1).
func ProximaTentativa(
	n int,
	agora time.Time,
	sorte func() float64, //nolint:revive // o acaso da simulação
) time.Time {
	espera := time.Duration(1<<min(n, 10)) * time.Second
	return agora.Add(espera)
}

// livro:fim missao-04
