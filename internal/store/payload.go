package store

import (
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// Payload é a parte de um evento que vai para a coluna payload, em
// JSON: os campos que só alguns tipos de evento preenchem.
type Payload struct {
	Worker  string    `json:"worker,omitempty"`
	Cause   string    `json:"cause,omitempty"`
	RunAt   time.Time `json:"run_at,omitzero"`
	Attempt int       `json:"attempt,omitempty"`
}

// PayloadOf extrai o payload de um evento.
func PayloadOf(e job.Event) Payload {
	return Payload{
		Worker:  e.Worker,
		Cause:   e.Cause,
		RunAt:   e.RunAt,
		Attempt: e.Attempt,
	}
}

// Event remonta o evento a partir do tipo, do instante e do payload.
func (p Payload) Event(t job.EventType, at time.Time) job.Event {
	return job.Event{
		Type:    t,
		At:      at.UTC(),
		Worker:  p.Worker,
		Cause:   p.Cause,
		RunAt:   p.RunAt.UTC(),
		Attempt: p.Attempt,
	}
}
