package job

import (
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
)

// Spec é o pedido de enfileiramento, como chega da aplicação.
type Spec struct {
	ID          id.JobID
	Namespace   string
	Queue       string
	Kind        string
	Args        []byte    // JSON
	UniqueKey   string    // vazio: sem deduplicação (Cap. 12 e 16)
	OrderingKey string    // mesma chave: um por vez, em ordem
	Priority    int       // 1 (mais alta) a 4; zero vale 2
	MaxAttempts int       // zero vale 25
	RunAt       time.Time // zero: agora
	TraceParent string    // contexto W3C de quem enfileirou (Cap. 30)
}

// Job é a projeção do estado corrente, derivada do histórico.
type Job struct {
	ID          id.JobID
	Namespace   string
	Queue       string
	Kind        string
	Args        []byte
	UniqueKey   string
	OrderingKey string
	Priority    int
	State       State
	Attempt     int
	MaxAttempts int
	ScheduledAt time.Time
	AttemptedAt time.Time
	AttemptedBy string
	HeartbeatAt time.Time // último batimento da tentativa corrente
	FinalizedAt time.Time
	LastError   string
	TraceParent string // contexto W3C de quem enfileirou (Cap. 30)
}

const (
	prioridadePadrao = 2
	tentativasPadrao = 25
	prioridadeMinima = 1
	prioridadeMaxima = 4
)

// livro:inicio particao-do-job

// Particao é a partição do job: a da chave de ordem, se houver — todos
// os jobs da chave no mesmo dono —, ou a do próprio id.
func (j Job) Particao() int {
	if j.OrderingKey != "" {
		return id.Particao(j.OrderingKey)
	}
	return id.Particao(j.ID.String())
}

// livro:fim particao-do-job
