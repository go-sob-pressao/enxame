package webhook

import (
	"slices"
	"time"
)

// Mensagem é um evento publicado pela aplicação, a entregar a cada
// endpoint inscrito no tipo dele.
type Mensagem struct {
	ID        string
	Namespace string
	EventType string
	Payload   []byte // JSON
	CriadaEm  time.Time
}

// Tentativa é o registro de uma tentativa de entrega.
type Tentativa struct {
	MessageID  string
	EndpointID string
	Attempt    int
	JobID      string
	Status     int // 0: nem houve resposta
	Duracao    time.Duration
	Erro       string
	Trecho     string // o começo da resposta, para diagnóstico
	Em         time.Time
}

// Inscrito diz se o endpoint recebe mensagens do tipo dado: sem tipos
// declarados, recebe todos.
func Inscrito(e Endpoint, tipo string) bool {
	return !e.Disabled &&
		(len(e.EventTypes) == 0 || slices.Contains(e.EventTypes, tipo))
}
