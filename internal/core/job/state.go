package job

// State é o estado corrente do job. Os valores são os mesmos da coluna
// job.state no banco.
type State string

// livro:inicio estados

// Estados de um job.
const (
	StateScheduled State = "scheduled" // agendado para o futuro
	StateAvailable State = "available" // elegível para um worker
	StateRunning   State = "running"   // tentativa em andamento
	StateRetryable State = "retryable" // falhou; aguarda o backoff
	StateCompleted State = "completed" // final: sucesso
	// final: tentativas esgotadas ou erro permanente
	StateDiscarded State = "discarded"
	StateCancelled State = "cancelled" // final: cancelado
)

// livro:fim estados

// Final informa se o estado não admite mais transições.
func (s State) Final() bool {
	return s == StateCompleted || s == StateDiscarded ||
		s == StateCancelled
}
