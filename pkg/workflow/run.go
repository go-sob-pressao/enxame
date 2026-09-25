package workflow

import "encoding/json"

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
