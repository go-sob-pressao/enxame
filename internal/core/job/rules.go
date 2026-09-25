package job

import "slices"

// livro:inicio origens

// origens diz de quais estados cada decisão pode partir: a mesma
// tabela da especificação, agora do lado do domínio. As funções de
// decisão a consultam, em vez de repetir a condição cada uma à sua
// maneira. Uma decisão fora da tabela não parte de estado nenhum.
var origens = map[string][]State{
	"iniciar":           {StateAvailable},
	"concluir":          {StateRunning},
	"registrar falha":   {StateRunning},
	"tornar disponível": {StateScheduled, StateRetryable},
	"resgatar":          {StateRunning},
	"cancelar": {
		StateScheduled, StateAvailable, StateRetryable,
	},
}

// exigir recusa a decisão se ela não pode partir do estado do job.
func exigir(j Job, acao string) error {
	if !slices.Contains(origens[acao], j.State) {
		return invalida(j, acao)
	}
	return nil
}

// livro:fim origens
