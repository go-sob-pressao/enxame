package id

import (
	"strconv"
	"time"
)

// livro:inicio idempotencia

// As chaves de idempotência são derivadas do que identifica o trabalho,
// e não da tentativa: todas as tentativas do mesmo trabalho chegam ao
// sistema externo com a mesma chave, e ele pode reconhecer a repetição.

// JobKey é a chave de um job: o namespace separa aplicações que
// compartilham o Enxame, e o id do job não muda entre tentativas.
func JobKey(namespace string, j JobID) string {
	return "job:" + namespace + ":" + j.String()
}

// StepKey é a chave de um passo de workflow: a posição no histórico do
// run, que o replay nunca muda.
func StepKey(runID string, seq int) string {
	return "step:" + runID + ":" + strconv.Itoa(seq)
}

// WindowKey é a chave de um disparo de agendamento: o instante
// previsto, e não o instante em que o disparo de fato aconteceu.
func WindowKey(
	namespace, scheduleID string,
	previsto time.Time,
) string {
	return "cron:" + namespace + ":" + scheduleID + ":" +
		previsto.UTC().Format(time.RFC3339)
}

// livro:fim idempotencia
