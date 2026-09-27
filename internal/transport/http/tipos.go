package http

import (
	"encoding/json/jsontext"
	"time"
)

// NovoJob é o pedido de enfileiramento.
type NovoJob struct {
	Queue       string         `json:"queue"`
	Kind        string         `json:"kind"`
	Args        jsontext.Value `json:"args,omitzero"`
	UniqueKey   string         `json:"unique_key,omitzero"`
	OrderingKey string         `json:"ordering_key,omitzero"`
	RunAt       time.Time      `json:"run_at,omitzero"`
	MaxAttempts int            `json:"max_attempts,omitzero"`
}

// Job é o job como a API o mostra.
type Job struct {
	ID          string         `json:"id"`
	Queue       string         `json:"queue"`
	Kind        string         `json:"kind"`
	Args        jsontext.Value `json:"args"`
	State       string         `json:"state"`
	Attempt     int            `json:"attempt"`
	MaxAttempts int            `json:"max_attempts"`
	ScheduledAt time.Time      `json:"scheduled_at"`
	LastError   string         `json:"last_error,omitzero"`
	FinalizedAt time.Time      `json:"finalized_at,omitzero"`
}

// NovoWorkflow é o pedido de início de um run.
type NovoWorkflow struct {
	Type       string         `json:"type"`
	WorkflowID string         `json:"workflow_id"`
	Input      jsontext.Value `json:"input,omitzero"`
}

// Run é um run de workflow como a API o mostra.
type Run struct {
	ID     string         `json:"id"`
	State  string         `json:"state"`
	Output jsontext.Value `json:"output,omitzero"`
	Error  string         `json:"error,omitzero"`
}

// NovoEndpoint é o pedido de inscrição de um endpoint de webhook.
type NovoEndpoint struct {
	URL         string   `json:"url"`
	Description string   `json:"description,omitzero"`
	EventTypes  []string `json:"event_types"`
	SecretRef   string   `json:"secret_ref"`
	RateLimit   int      `json:"rate_limit,omitzero"`
}

// Endpoint é o endpoint como a API o mostra.
type Endpoint struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	EventTypes  []string  `json:"event_types"`
	SecretRef   string    `json:"secret_ref"`
	RateLimit   int       `json:"rate_limit"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// Lista embrulha listas, para que a resposta seja sempre um objeto e
// possa ganhar campos sem quebrar clientes.
type Lista[T any] struct {
	Items []T `json:"items"`
}
