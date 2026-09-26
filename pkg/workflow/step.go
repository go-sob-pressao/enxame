package workflow

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/pkg/job"
)

// Permanent marca o erro de um passo como definitivo: ele é gravado, e
// o replay o devolve sem executar o passo de novo. Um erro sem a marca
// não é gravado — o passo roda outra vez na próxima execução.
func Permanent(err error) error { return permanente{err} }

type permanente struct{ error }

func (p permanente) Unwrap() error { return p.error }

// livro:inicio step

// Step executa fn uma única vez por run. Na primeira execução, roda fn
// e grava o resultado; em cada replay, devolve o resultado gravado sem
// chamar fn. O nome é conferido: se o código mudou a ordem dos passos,
// o replay para com NonDeterministicError em vez de devolver o
// resultado de um passo como se fosse de outro.
func Step[T any](
	c *Context,
	name string,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T
	r, gravado, err := c.proximo(name, KindCall)
	if err != nil {
		return zero, err
	}
	if gravado {
		if r.Err != "" {
			return zero, &StepError{Step: name, Msg: r.Err}
		}
		var v T
		err := json.Unmarshal(r.Output, &v)
		return v, err
	}
	// Dentro do passo, job.IdempotencyKey devolve a chave da posição:
	// a mesma em todos os replays que chegarem a executá-la.
	v, err := fn(job.WithInfo(c.ctx, job.Info{
		Kind: name, IdempotencyKey: id.StepKey(c.runID, r.Seq)}))
	var p permanente
	switch {
	case errors.As(err, &p):
		r.Err = p.Error()
		if err := c.hist.Append(c.ctx, r); err != nil {
			return zero, err
		}
		return zero, &StepError{Step: name, Msg: r.Err}
	case err != nil:
		return zero, err // não gravado: roda de novo depois
	}
	if r.Output, err = json.Marshal(v); err != nil {
		return zero, err
	}
	return v, c.hist.Append(c.ctx, r)
}

// livro:fim step
