package runner

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// PanicError é o pânico de um handler, convertido em erro da tentativa.
type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("pânico no handler: %v", e.Value)
}

// livro:inicio recover

// Call executa o handler e converte pânico em erro. O recover fica em
// volta de UMA tentativa: um job defeituoso falha sozinho, sem derrubar
// o worker nem os outros jobs — e o pânico vira histórico, com a pilha.
func Call(ctx context.Context, h Handler, j job.Job) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = &PanicError{Value: v, Stack: debug.Stack()}
		}
	}()
	return h(ctx, j)
}

// livro:fim recover
