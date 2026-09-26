package poller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
)

// livro:inicio remoto

// Remoto é a fila do pool do lado de um worker remoto: a mesma
// interface da fila em memória e da fila no Postgres, agora sobre um
// stream gRPC. O pool da Parte I não sabe a diferença.
type Remoto struct {
	ctx      context.Context
	cliente  enxamev1.WorkerServiceClient
	stream   enxamev1.WorkerService_FetchClient
	jobs     chan job.Job
	mu       sync.Mutex
	aviso    chan struct{}
	tentativ map[id.JobID]int
	falha    error
}

// Conectar abre o stream de busca. capacidade é quantos jobs o worker
// aceita ter recebido e ainda não começado.
func Conectar(
	ctx context.Context,
	conn *grpc.ClientConn,
	worker string,
	filas []string,
	capacidade int,
) (*Remoto, error) {
	c := enxamev1.NewWorkerServiceClient(conn)
	stream, err := c.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	credito := int32(capacidade) //nolint:gosec // capacidade pequena
	err = stream.Send(&enxamev1.FetchRequest{Worker: worker,
		Queues: filas, Credit: credito})
	if err != nil {
		return nil, err
	}
	r := &Remoto{ctx: ctx, cliente: c, stream: stream,
		jobs:     make(chan job.Job, capacidade),
		aviso:    make(chan struct{}),
		tentativ: map[id.JobID]int{}}
	go r.receber()
	return r, nil
}

// receber põe cada job recebido na fila local e avisa o pool.
func (r *Remoto) receber() {
	for {
		m, err := r.stream.Recv()
		if err != nil {
			r.mu.Lock()
			r.falha = fmt.Errorf("stream de busca: %w", err)
			close(r.aviso)
			r.mu.Unlock()
			return
		}
		j, err := deProto(m.GetJob())
		if err != nil {
			continue
		}
		r.mu.Lock()
		r.tentativ[j.ID] = j.Attempt
		r.mu.Unlock()
		r.jobs <- j
		r.mu.Lock()
		close(r.aviso)
		r.aviso = make(chan struct{})
		r.mu.Unlock()
	}
}

// Fetch entrega um job já recebido, se houver, e devolve um crédito ao
// motor. A fila e o worker vêm do stream, não da chamada.
func (r *Remoto) Fetch(
	string, time.Time, string,
) (job.Job, bool, error) {
	select {
	case j := <-r.jobs:
		err := r.stream.Send(&enxamev1.FetchRequest{Credit: 1})
		return j, true, err
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return job.Job{}, false, r.falha
}

// livro:fim remoto

// Changed devolve o canal fechado quando chegar um job.
func (r *Remoto) Changed() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.aviso
}

// Complete registra o sucesso da tentativa.
func (r *Remoto) Complete(jid id.JobID, _ time.Time) error {
	return r.terminar(jid, &enxamev1.FinishRequest{})
}

// Fail registra a falha da tentativa.
func (r *Remoto) Fail(
	jid id.JobID,
	at time.Time,
	cause string,
	permanent bool,
	retryAt time.Time,
) error {
	return r.terminar(jid, &enxamev1.FinishRequest{Error: cause,
		Permanent: permanent,
		RetryIn:   durationpb.New(retryAt.Sub(at))})
}

func (r *Remoto) terminar(
	jid id.JobID,
	req *enxamev1.FinishRequest,
) error {
	r.mu.Lock()
	tentativa := r.tentativ[jid]
	delete(r.tentativ, jid)
	r.mu.Unlock()
	req.JobId = jid.String()
	req.Attempt = int32(tentativa) //nolint:gosec // cabe
	return repetir(r.ctx, func() error {
		_, err := r.cliente.Finish(r.ctx, req)
		return err
	})
}

// livro:inicio repetir

// repetir tenta de novo o que falhou por motivo passageiro — a rede, o
// prazo, um conflito —, até três vezes. Se ainda assim não der, desiste
// sem erro: o job continua em execução no motor, sem batimento, e o
// resgate o devolve à fila. Derrubar o pool por causa de uma rede
// instável transformaria uma falha de um job em falha de todos.
func repetir(ctx context.Context, f func() error) error {
	espera := 100 * time.Millisecond
	for tentativa := 1; ; tentativa++ {
		err := f()
		switch status.Code(err) {
		case codes.OK, codes.NotFound:
			return nil // feito, ou o job nem existe mais
		case codes.Unavailable, codes.DeadlineExceeded,
			codes.Aborted, codes.Internal:
			if tentativa == 3 {
				return nil // o resgate decide
			}
		default: // perdida, sem autenticação: quem chama decide
			return err
		}
		select {
		case <-time.After(espera):
			espera *= 2
		case <-ctx.Done():
			return nil
		}
	}
}

// livro:fim repetir

// Heartbeat diz ao motor que a tentativa continua viva.
func (r *Remoto) Heartbeat(
	ctx context.Context,
	jid id.JobID,
	attempt int,
) error {
	_, err := r.cliente.Heartbeat(ctx, &enxamev1.HeartbeatRequest{
		JobId: jid.String(), Attempt: int32(attempt)}) //nolint:gosec
	return err
}

// Promote não faz nada: no modo servidor, quem promove é o motor.
func (*Remoto) Promote(time.Time) (int, error) { return 0, nil }

func deProto(m *enxamev1.Job) (job.Job, error) {
	jid, err := id.ParseJobID(m.GetId())
	if err != nil {
		return job.Job{}, err
	}
	return job.Job{ID: jid, Namespace: m.GetNamespace(),
		Queue: m.GetQueue(), Kind: m.GetKind(), Args: m.GetArgs(),
		Attempt:     int(m.GetAttempt()),
		MaxAttempts: int(m.GetMaxAttempts()),
		State:       job.StateRunning}, nil
}
