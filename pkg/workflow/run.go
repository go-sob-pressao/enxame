package workflow

import (
	"encoding/json"
	"time"
)

// RunState é o estado de um run.
type RunState string

// Os estados de um run.
const (
	RunRunning   RunState = "running"
	RunCompleted RunState = "completed"
	RunFailed    RunState = "failed"
)

// Run é uma execução de um workflow: o tipo, a entrada e, quando
// encerrada, a saída ou o erro. O histórico de passos fica à parte.
type Run struct {
	ID         string
	Namespace  string
	WorkflowID string // id de negócio, escolhido pela aplicação
	Type       string
	Input      json.RawMessage
	State      RunState
	Output     json.RawMessage
	Err        string
}

// Continuation é o pedido de uma nova execução do run, a partir de At,
// para a posição Seq do histórico. O runtime a grava junto com o passo
// que a exigiu, na mesma transação: ou os dois existem, ou nenhum.
type Continuation struct {
	Seq int
	At  time.Time
}

// O job que avança um run: um por posição do histórico.
const (
	AdvanceKind  = "workflow.avancar"
	AdvanceQueue = "workflow"
)

// AdvanceArgs são os argumentos do job que avança um run.
type AdvanceArgs struct {
	RunID string `json:"run_id"`
	Seq   int    `json:"seq"`
}
