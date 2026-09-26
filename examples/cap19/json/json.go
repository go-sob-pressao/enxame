// Package json mostra as armadilhas do JSON em Go e o que mudou do
// encoding/json para o encoding/json/v2.
package json

import "time"

// Pedido é o tipo dos exemplos.
type Pedido struct {
	ID       string            `json:"id"`
	Cliente  string            `json:"cliente"`
	Itens    []string          `json:"itens"`
	Extras   map[string]string `json:"extras"`
	Desconto int               `json:"desconto,omitempty"`
	Parcelas *int              `json:"parcelas,omitempty"`
	CriadoEm time.Time         `json:"criado_em"`
	Valor    int64             `json:"valor"`
}

// Resposta é o que a API devolve num benchmark.
type Resposta struct {
	ID          string    `json:"id"`
	Queue       string    `json:"queue"`
	Kind        string    `json:"kind"`
	State       string    `json:"state"`
	Attempt     int       `json:"attempt"`
	MaxAttempts int       `json:"max_attempts"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Tags        []string  `json:"tags"`
}
