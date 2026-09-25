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
	Priority    int       // 1 (mais alta) a 4; zero vale 2
	MaxAttempts int       // zero vale 25
	RunAt       time.Time // zero: agora
}

// Job é a projeção do estado corrente, derivada do histórico.
type Job struct {
	ID          id.JobID
	Namespace   string
	Queue       string
	Kind        string
	Args        []byte
	UniqueKey   string
	Priority    int
	State       State
	Attempt     int
	MaxAttempts int
	ScheduledAt time.Time
	AttemptedAt time.Time
	AttemptedBy string
	FinalizedAt time.Time
	LastError   string
}

const (
	prioridadePadrao = 2
	tentativasPadrao = 25
	prioridadeMinima = 1
	prioridadeMaxima = 4
)
