// Package schedule — agendamentos periódicos: a expressão cron, a
// próxima janela e a decisão de cada disparo, puras.
//
// Camada: internal/core
// Introduzido no livro: Cap. 16 e Cap. 17 — ver docs/mapa-capitulos.md
package schedule

import "time"

// Schedule é um agendamento periódico: a cada janela da expressão cron,
// um job do kind dado.
type Schedule struct {
	Namespace string
	ID        string
	Expr      string
	Timezone  string
	Queue     string
	Kind      string
	Args      []byte
	NextFire  time.Time // a próxima janela prevista
}
