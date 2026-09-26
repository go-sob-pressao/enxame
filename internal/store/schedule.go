package store

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
