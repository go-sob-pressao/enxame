package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Kind é o tipo de um passo gravado no histórico.
type Kind string

// Os tipos de passo desta versão. O sinal é do Capítulo 17.
const (
	KindCall       Kind = "call"
	KindSleep      Kind = "sleep"
	KindSideEffect Kind = "side_effect"
)

// Record é um passo gravado: a memória que o replay consulta.
type Record struct {
	Seq    int
	Name   string
	Kind   Kind
	Output json.RawMessage // resultado de call e side_effect
	Err    string          // erro definitivo de call
	WakeAt time.Time       // hora de acordar de sleep
}

// History é o histórico de passos de um run, visto pelo workflow.
// Lookup devolve o passo seq, se já foi gravado; Append grava o
// próximo.
type History interface {
	Lookup(seq int) (Record, bool)
	Append(ctx context.Context, r Record) error
}

// Context é o que a função de workflow recebe. Cada chamada a Step,
// Sleep, SideEffect ou Now ocupa a próxima posição do histórico.
type Context struct {
	ctx     context.Context
	runID   string
	hist    History
	seq     int
	clock   func() time.Time
	posicao int
}

// NewContext é usado pelo runtime que executa o workflow. Com posicao
// zero, a execução avança por quantos passos novos encontrar. Com
// posicao maior que zero, só a posição dada pode ser executada pela
// primeira vez: ao chegar a outra posição nova, a execução para, e o
// job daquela posição a executará.
func NewContext(
	ctx context.Context,
	runID string,
	h History,
	clock func() time.Time,
	posicao int,
) *Context {
	return &Context{ctx: ctx, runID: runID, hist: h, clock: clock,
		posicao: posicao}
}

// Context devolve o context.Context da execução, para cancelamento.
func (c *Context) Context() context.Context { return c.ctx }

// livro:inicio determinismo

// próximo avança a posição e devolve o passo gravado nela, conferindo
// que o código pede, nessa posição, o mesmo passo que pediu antes.
func (c *Context) proximo(name string, k Kind) (Record, bool, error) {
	c.seq++
	r, ok := c.hist.Lookup(c.seq)
	if !ok && c.posicao > 0 && c.seq != c.posicao {
		// Outra posição nova: é trabalho de outro job.
		return r, false, &SuspendedError{Until: c.clock()}
	}
	if !ok {
		return Record{Seq: c.seq, Name: name, Kind: k}, false, nil
	}
	if r.Name != name || r.Kind != k {
		return r, true, &NonDeterministicError{
			Seq: c.seq, Recorded: r.Name, RecordedKind: r.Kind,
			Requested: name, RequestedKind: k,
		}
	}
	return r, true, nil
}

// livro:fim determinismo

// ErrSuspended indica que o workflow parou à espera de algo — o fim de
// um Sleep. Não é falha: o runtime reexecuta o workflow mais tarde.
var ErrSuspended = errors.New("workflow suspenso")

// SuspendedError diz até quando o workflow está suspenso.
type SuspendedError struct{ Until time.Time }

func (e *SuspendedError) Error() string {
	return fmt.Sprintf("workflow suspenso até %s",
		e.Until.Format(time.RFC3339Nano))
}

// Is faz errors.Is(err, ErrSuspended) valer para SuspendedError.
func (e *SuspendedError) Is(alvo error) bool {
	return alvo == ErrSuspended
}

// NonDeterministicError indica que o código do workflow, reexecutado,
// pediu na posição Seq um passo diferente do que está gravado. O run
// não pode avançar: o histórico e o código contam histórias diferentes.
type NonDeterministicError struct {
	Seq           int
	Recorded      string
	RecordedKind  Kind
	Requested     string
	RequestedKind Kind
}

func (e *NonDeterministicError) Error() string {
	return fmt.Sprintf(
		"passo %d: histórico tem %s %q, código pediu %s %q",
		e.Seq, e.RecordedKind, e.Recorded,
		e.RequestedKind, e.Requested)
}

// StepError é o erro definitivo de um passo, devolvido de novo em cada
// replay.
type StepError struct {
	Step string
	Msg  string
}

func (e *StepError) Error() string {
	return fmt.Sprintf("passo %q: %s", e.Step, e.Msg)
}
