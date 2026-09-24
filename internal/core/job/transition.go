package job

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidTransition indica uma decisão impossível no estado
// corrente.
var ErrInvalidTransition = errors.New("transição inválida")

func invalida(j Job, acao string) error {
	return fmt.Errorf(
		"%w: %s com o job %s em %q",
		ErrInvalidTransition,
		acao,
		j.ID,
		j.State,
	)
}

// As funções abaixo DECIDEM: recebem a projeção e o instante, e
// devolvem os eventos a gravar. Nenhuma lê o relógio, gera
// identificador ou faz I/O. Apply projeta os eventos; decidir e
// projetar são separados para que o histórico seja a fonte da verdade
// (ADR-001, Cap. 14).

// livro:inicio decidir

// Insert valida o pedido e devolve o primeiro evento do job.
func Insert(s Spec, at time.Time) ([]Event, Job, error) {
	if s.ID.IsZero() || s.Kind == "" || s.Queue == "" {
		return nil, Job{}, errors.New("job sem id, kind ou fila")
	}
	j := Job{
		ID: s.ID, Namespace: s.Namespace, Queue: s.Queue,
		Kind: s.Kind, Args: s.Args,
		Priority: s.Priority, MaxAttempts: s.MaxAttempts,
	}
	if j.Priority == 0 {
		j.Priority = prioridadePadrao
	}
	if j.Priority < prioridadeMinima || j.Priority > prioridadeMaxima {
		return nil, Job{}, fmt.Errorf(
			"prioridade %d fora de 1..4",
			j.Priority,
		)
	}
	if j.MaxAttempts == 0 {
		j.MaxAttempts = tentativasPadrao
	}
	evs := []Event{{Type: EventInserted, At: at}}
	if s.RunAt.After(at) {
		evs = append(
			evs,
			Event{Type: EventScheduled, At: at, RunAt: s.RunAt},
		)
	}
	return evs, j, nil
}

// Start registra que um worker pegou o job disponível.
func Start(j Job, at time.Time, worker string) ([]Event, error) {
	if j.State != StateAvailable {
		return nil, invalida(j, "iniciar")
	}
	return []Event{
		{
			Type:    EventAttemptStart,
			At:      at,
			Worker:  worker,
			Attempt: j.Attempt + 1,
		},
	}, nil
}

// Complete registra que o handler devolveu nil.
func Complete(j Job, at time.Time) ([]Event, error) {
	if j.State != StateRunning {
		return nil, invalida(j, "concluir")
	}
	return []Event{{Type: EventCompleted, At: at}}, nil
}

// Fail registra a falha do handler. Com tentativas restantes e erro não
// permanente, o job volta em retryAt; senão, é descartado.
func Fail(
	j Job,
	at time.Time,
	cause string,
	permanent bool,
	retryAt time.Time,
) ([]Event, error) {
	if j.State != StateRunning {
		return nil, invalida(j, "registrar falha")
	}
	evs := []Event{
		{
			Type:    EventAttemptFailed,
			At:      at,
			Cause:   cause,
			Attempt: j.Attempt,
		},
	}
	if permanent || j.Attempt >= j.MaxAttempts {
		return append(
			evs,
			Event{Type: EventDiscarded, At: at, Cause: cause},
		), nil
	}
	return append(
		evs,
		Event{Type: EventScheduled, At: at, RunAt: retryAt},
	), nil
}

// MakeAvailable torna elegível um job agendado ou em espera de retry
// cuja hora chegou.
func MakeAvailable(j Job, at time.Time) ([]Event, error) {
	if j.State != StateScheduled && j.State != StateRetryable {
		return nil, invalida(j, "tornar disponível")
	}
	if at.Before(j.ScheduledAt) {
		return nil, fmt.Errorf(
			"%w: job %s só fica elegível em %s",
			ErrInvalidTransition,
			j.ID,
			j.ScheduledAt,
		)
	}
	return []Event{{Type: EventMadeAvailable, At: at}}, nil
}

// Cancel cancela o job — só antes de uma tentativa começar.
func Cancel(j Job, at time.Time) ([]Event, error) {
	switch j.State {
	case StateScheduled, StateAvailable, StateRetryable:
		return []Event{{Type: EventCancelled, At: at}}, nil
	default:
		return nil, invalida(j, "cancelar")
	}
}

// Rescue devolve à fila o job cujo worker morreu no meio da tentativa.
func Rescue(j Job, at time.Time) ([]Event, error) {
	if j.State != StateRunning {
		return nil, invalida(j, "resgatar")
	}
	return []Event{{Type: EventRescued, At: at}}, nil
}

// livro:fim decidir

// livro:inicio apply

// Apply projeta um evento sobre o job. É a única função que muda a
// projeção.
func Apply(j Job, e Event) (Job, error) {
	switch e.Type {
	case EventInserted:
		j.State, j.ScheduledAt = StateAvailable, e.At
	case EventScheduled:
		j.ScheduledAt = e.RunAt
		if j.Attempt > 0 {
			j.State = StateRetryable
		} else {
			j.State = StateScheduled
		}
	case EventMadeAvailable:
		j.State = StateAvailable
	case EventAttemptStart:
		j.State, j.Attempt = StateRunning, e.Attempt
		j.AttemptedAt, j.AttemptedBy = e.At, e.Worker
	case EventHeartbeat:
	case EventAttemptFailed:
		j.LastError = e.Cause
	case EventCompleted:
		j.State, j.FinalizedAt = StateCompleted, e.At
	case EventDiscarded:
		j.State, j.FinalizedAt = StateDiscarded, e.At
	case EventCancelled:
		j.State, j.FinalizedAt = StateCancelled, e.At
	case EventRescued:
		j.State = StateAvailable
	default:
		return j, fmt.Errorf("evento desconhecido: %d", e.Type)
	}
	return j, nil
}

// livro:fim apply

// ApplyAll projeta uma sequência de eventos.
func ApplyAll(j Job, evs []Event) (Job, error) {
	var err error
	for _, e := range evs {
		if j, err = Apply(j, e); err != nil {
			return j, err
		}
	}
	return j, nil
}
