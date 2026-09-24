package job

import "time"

// EventType identifica o fato registrado no histórico. Os números são
// os da coluna job_event.event_type e não podem mudar
// (docs/protocol/eventos.md).
type EventType uint8

// livro:inicio eventos

// Tipos de evento do histórico de um job.
const (
	EventInserted EventType = 1 // enfileirado
	// agendado para o futuro, ou retry com backoff
	EventScheduled     EventType = 2
	EventMadeAvailable EventType = 3 // o timer o tornou elegível
	EventAttemptStart  EventType = 4 // um worker o pegou
	// job longo sinalizou que está vivo
	EventHeartbeat EventType = 5
	// erro, pânico recuperado ou timeout
	EventAttemptFailed EventType = 6
	EventCompleted     EventType = 7 // concluído
	// tentativas esgotadas ou erro permanente
	EventDiscarded EventType = 8
	EventCancelled EventType = 9  // cancelado
	EventRescued   EventType = 10 // tentativa órfã devolvida à fila
)

// livro:fim eventos

var nomesDeEvento = [...]string{
	EventInserted: "JobInserted", EventScheduled: "JobScheduled",
	EventMadeAvailable: "JobMadeAvailable",
	EventAttemptStart:  "AttemptStarted",
	EventHeartbeat:     "AttemptHeartbeat",
	EventAttemptFailed: "AttemptFailed",
	EventCompleted:     "JobCompleted", EventDiscarded: "JobDiscarded",
	EventCancelled: "JobCancelled", EventRescued: "JobRescued",
}

func (t EventType) String() string {
	if int(t) < len(nomesDeEvento) && nomesDeEvento[t] != "" {
		return nomesDeEvento[t]
	}
	return "EventType(desconhecido)"
}

// Event é um fato do histórico. Só os campos do tipo correspondente são
// preenchidos.
type Event struct {
	Type    EventType
	At      time.Time
	Worker  string    // AttemptStarted
	Cause   string    // AttemptFailed, JobDiscarded
	RunAt   time.Time // JobScheduled: quando fica elegível
	Attempt int       // AttemptStarted, AttemptFailed
}
