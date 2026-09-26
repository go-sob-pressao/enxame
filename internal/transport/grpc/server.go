package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
)

// Motor é o que o servidor precisa do armazenamento.
type Motor interface {
	Claim(ctx context.Context, queue string, at time.Time,
		worker string) (job.Job, bool, error)
	Heartbeat(ctx context.Context, jid id.JobID, at time.Time,
		attempt int) error
	Decide(ctx context.Context, jid id.JobID,
		decidir func(job.Job) ([]job.Event, error)) (job.Job, error)
}

// Server atende os workers remotos.
type Server struct {
	enxamev1.UnimplementedWorkerServiceServer
	Motor Motor
	Poll  time.Duration // intervalo da busca quando não há job
	Now   func() time.Time
}

// livro:inicio fetch

// Fetch mantém um stream por worker. Cada crédito recebido vale um
// job: o servidor busca nas filas do worker, envia, e só busca de novo
// quando houver crédito. Sem job disponível, espera Poll e tenta outra
// vez — o long-poll do Capítulo 4, agora do lado do motor, com o worker
// do outro lado da rede.
func (s *Server) Fetch(
	stream enxamev1.WorkerService_FetchServer,
) error {
	ctx := stream.Context()
	primeiro, err := stream.Recv()
	if err != nil {
		return err
	}
	worker, filas := primeiro.GetWorker(), primeiro.GetQueues()
	if worker == "" || len(filas) == 0 {
		return fmt.Errorf("%w: worker e filas são obrigatórios",
			job.ErrInvalidTransition)
	}
	creditos := make(chan int32)
	go func() { // lê os créditos até o worker fechar o stream
		defer close(creditos)
		for {
			m, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case creditos <- m.GetCredit():
			case <-ctx.Done():
				return
			}
		}
	}()
	credito := primeiro.GetCredit()
	for {
		for credito > 0 {
			j, ok, err := s.buscar(ctx, filas, worker)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			if err := stream.Send(&enxamev1.FetchResponse{
				Job: paraProto(j)}); err != nil {
				return err
			}
			credito--
		}
		select {
		case c, ok := <-creditos:
			if !ok {
				return nil // o worker foi embora
			}
			credito += c
		case <-time.After(s.Poll):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// livro:fim fetch

func (s *Server) buscar(
	ctx context.Context,
	filas []string,
	worker string,
) (job.Job, bool, error) {
	for _, f := range filas {
		j, ok, err := s.Motor.Claim(ctx, f, s.Now(), worker)
		if err != nil || ok {
			return j, ok, err
		}
	}
	return job.Job{}, false, nil
}

// Heartbeat renova a tentativa.
func (s *Server) Heartbeat(
	ctx context.Context,
	req *enxamev1.HeartbeatRequest,
) (*enxamev1.HeartbeatResponse, error) {
	jid, err := id.ParseJobID(req.GetJobId())
	if err != nil {
		return nil, err
	}
	return &enxamev1.HeartbeatResponse{}, s.Motor.Heartbeat(ctx, jid,
		s.Now(), int(req.GetAttempt()))
}

// livro:inicio finish

// Finish registra o fim de uma tentativa. O número da tentativa vem na
// requisição e é conferido: o fim atrasado de uma tentativa resgatada
// não conclui, nem falha, a tentativa que a substituiu.
func (s *Server) Finish(
	ctx context.Context,
	req *enxamev1.FinishRequest,
) (*enxamev1.FinishResponse, error) {
	jid, err := id.ParseJobID(req.GetJobId())
	if err != nil {
		return nil, err
	}
	agora := s.Now()
	_, err = s.Motor.Decide(ctx, jid,
		func(j job.Job) ([]job.Event, error) {
			if j.Attempt != int(req.GetAttempt()) {
				return nil, fmt.Errorf("%w: fim da tentativa %d; a "+
					"atual é %d", job.ErrInvalidTransition,
					req.GetAttempt(), j.Attempt)
			}
			if req.GetError() == "" {
				return job.Complete(j, agora)
			}
			return job.Fail(j, agora, req.GetError(),
				req.GetPermanent(),
				agora.Add(req.GetRetryIn().AsDuration()))
		})
	return &enxamev1.FinishResponse{}, err
}

// livro:fim finish

func paraProto(j job.Job) *enxamev1.Job {
	return &enxamev1.Job{Id: j.ID.String(), Namespace: j.Namespace,
		Queue: j.Queue, Kind: j.Kind, Args: j.Args,
		Attempt:     int32(j.Attempt),     //nolint:gosec // cabe
		MaxAttempts: int32(j.MaxAttempts), //nolint:gosec // cabe
	}
}
