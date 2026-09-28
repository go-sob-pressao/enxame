package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/observ/metrics"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio busca

// Claim reserva o próximo job disponível da fila para worker, numa
// transação: trava a linha com FOR UPDATE SKIP LOCKED — outros workers,
// ao mesmo tempo, pulam a linha travada e pegam a seguinte, em vez de
// esperar por ela —, decide pelo domínio e grava a projeção e o evento.
func (s *Store) Claim(
	ctx context.Context,
	queue string,
	at time.Time,
	worker string,
) (job.Job, bool, error) {
	var reservado job.Job
	achou := false
	at, err := s.instante(ctx, at)
	if err != nil {
		return job.Job{}, false, err
	}
	if s.dono != nil {
		return s.claimDoDono(ctx, queue, at, worker)
	}
	err = s.transacao(ctx, func(tx pgx.Tx) error {
		j, v, err := ler(tx.QueryRow(ctx, `SELECT `+colunas+` FROM job
			WHERE queue = $1 AND state = 'available' AND `+cabeca+`
			ORDER BY priority, scheduled_at, job_id
			LIMIT 1 FOR UPDATE SKIP LOCKED`, queue))
		if errors.Is(err, store.ErrNotFound) {
			return nil // fila vazia, ou tudo travado por outros
		}
		if err != nil {
			return err
		}
		evs, err := job.Start(j, at, worker)
		if err != nil {
			return err
		}
		if reservado, err = job.ApplyAll(j, evs); err != nil {
			return err
		}
		achou = true
		return gravar(ctx, tx, reservado, evs, v)
	})
	if achou && err == nil {
		metrics.JobReservado(reservado.Queue, reservado.ScheduledAt,
			reservado.AttemptedAt)
	}
	return reservado, achou, err
}

// livro:fim busca

// livro:inicio claim-do-dono

// claimDoDono reserva só das partições do nó, e confere, na mesma
// transação, o token da partição do job que achou. Um token recusado
// quer dizer que a partição tem outro dono: o nó é avisado, e a
// reserva volta vazia em vez de derrubar o pool.
func (s *Store) claimDoDono(
	ctx context.Context,
	queue string,
	at time.Time,
	worker string,
) (job.Job, bool, error) {
	tokens := s.dono.Tokens()
	if len(tokens) == 0 {
		return job.Job{}, false, nil
	}
	parts := make([]int32, 0, len(tokens))
	for p := range tokens {
		parts = append(parts, int32(p)) //nolint:gosec // < 512
	}
	var reservado job.Job
	achou, particao := false, -1
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, v, err := ler(tx.QueryRow(ctx, `SELECT `+colunas+` FROM job
			WHERE partition_id = ANY($2) AND queue = $1
			  AND state = 'available' AND `+cabeca+`
			ORDER BY priority, scheduled_at, job_id
			LIMIT 1 FOR UPDATE SKIP LOCKED`, queue, parts))
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		particao = j.Particao()
		err = ConferirCerca(ctx, tx, Cerca{Particao: particao,
			RangeID: tokens[particao]})
		if err != nil {
			return err
		}
		evs, err := job.Start(j, at, worker)
		if err != nil {
			return err
		}
		if reservado, err = job.ApplyAll(j, evs); err != nil {
			return err
		}
		achou = true
		return gravar(ctx, tx, reservado, evs, v)
	})
	if errors.Is(err, store.ErrCercado) {
		s.dono.Perdeu(particao)
		return job.Job{}, false, nil
	}
	if achou && err == nil {
		metrics.JobReservado(reservado.Queue, reservado.ScheduledAt,
			reservado.AttemptedAt)
	}
	return reservado, achou, err
}

// livro:fim claim-do-dono

// livro:inicio cabeca

// cabeca é a condição da ordem por chave: um job com ordering_key só
// pode ser reservado se não houver job mais antigo da mesma chave ainda
// por terminar — nem esperando, nem rodando, nem aguardando o retry. O
// primeiro da chave que falha segura os seguintes até dar certo ou ser
// descartado: é o preço da ordem.
const cabeca = `(ordering_key IS NULL OR NOT EXISTS (
	SELECT 1 FROM job anterior
	 WHERE anterior.namespace = job.namespace
	   AND anterior.ordering_key = job.ordering_key
	   AND anterior.state NOT IN ('completed', 'discarded', 'cancelled')
	   AND anterior.job_id < job.job_id))`

// livro:fim cabeca

// Decide aplica, numa transação e com a linha travada, uma decisão do
// domínio ao job jid: Complete, Fail, Cancel, Rescue.
func (s *Store) Decide(
	ctx context.Context,
	jid id.JobID,
	decidir func(job.Job) ([]job.Event, error),
) (job.Job, error) {
	var antes, novo job.Job
	err := s.transacao(ctx, func(tx pgx.Tx) error {
		j, v, err := ler(tx.QueryRow(ctx,
			`SELECT `+colunas+` FROM job WHERE job_id = $1 FOR UPDATE`,
			jid.String()))
		if err != nil {
			return err
		}
		antes = j
		evs, err := decidir(j)
		if err != nil {
			return err
		}
		if novo, err = job.ApplyAll(j, evs); err != nil {
			return err
		}
		return gravar(ctx, tx, novo, evs, v)
	})
	if err == nil {
		anotarFim(antes, novo)
	}
	return novo, err
}

// anotarFim conta o fim de uma tentativa: um job que estava rodando e
// deixou de estar.
func anotarFim(antes, depois job.Job) {
	if antes.State != job.StateRunning ||
		depois.State == job.StateRunning {
		return
	}
	resultado := "retry"
	switch depois.State {
	case job.StateCompleted:
		resultado = "concluido"
	case job.StateDiscarded, job.StateCancelled:
		resultado = "descartado"
	}
	fim := depois.FinalizedAt
	if fim.IsZero() {
		fim = time.Now()
	}
	metrics.JobTerminou(depois.Queue, resultado, antes.AttemptedAt, fim)
}

// Promote torna disponíveis, em lotes de até 100, os jobs agendados ou
// à espera de retry cuja hora chegou. Devolve quantos promoveu.
func (s *Store) Promote(
	ctx context.Context,
	at time.Time,
) (int, error) {
	at, err := s.instante(ctx, at)
	if err != nil {
		return 0, err
	}
	return s.emLote(ctx, `state IN ('scheduled', 'retryable')`+
		s.particaoDaCerca()+`
		AND scheduled_at <= $1`, at,
		func(j job.Job) ([]job.Event, error) {
			return job.MakeAvailable(j, at)
		})
}

// Heartbeat registra que a tentativa attempt do job continua viva.
func (s *Store) Heartbeat(
	ctx context.Context,
	jid id.JobID,
	at time.Time,
	attempt int,
) error {
	at, err := s.instante(ctx, at)
	if err != nil {
		return err
	}
	_, err = s.Decide(ctx, jid, func(j job.Job) ([]job.Event, error) {
		return job.Heartbeat(j, at, attempt)
	})
	return err
}

// livro:inicio resgate

// Rescue resgata, em lotes de até 100, os jobs em execução cujo último
// sinal de vida — o último batimento, ou o início da tentativa — é
// anterior a desde: o worker morreu, ou perdeu a conexão, sem
// registrar o fim. Uma tentativa longa que bate a tempo não é
// resgatada.
func (s *Store) Rescue(
	ctx context.Context,
	at, desde time.Time,
) (int, error) {
	agora, err := s.instante(ctx, at)
	if err != nil {
		return 0, err
	}
	// O prazo é o que o chamador pediu; o instante, o do relógio que
	// vale.
	at, desde = agora, agora.Add(-at.Sub(desde))
	return s.emLote(ctx, `state = 'running'`+s.particaoDaCerca()+`
		AND coalesce(heartbeat_at, attempted_at) < $1`,
		desde, func(j job.Job) ([]job.Event, error) {
			return job.Rescue(j, at)
		})
}

// livro:fim resgate

// emLote trava até 100 jobs que satisfazem onde, pulando os já
// travados, e aplica a cada um a decisão do domínio.
func (s *Store) emLote(
	ctx context.Context,
	onde string,
	arg any,
	decidir func(job.Job) ([]job.Event, error),
) (int, error) {
	n := 0
	var perdidas []int
	err := s.transacao(ctx, func(tx pgx.Tx) error {
		args := []any{arg}
		if s.dono != nil {
			validas, fora, err := conferirCercas(ctx, tx,
				s.dono.Tokens())
			if err != nil {
				return err
			}
			perdidas = fora
			onde += " AND partition_id = ANY($2)"
			args = append(args, validas)
		}
		rows, err := tx.Query(ctx, `SELECT `+colunas+` FROM job
			WHERE `+onde+` ORDER BY job_id LIMIT 100
			FOR UPDATE SKIP LOCKED`, args...)
		if err != nil {
			return err
		}
		type travado struct {
			j job.Job
			v int64
		}
		var lote []travado
		for rows.Next() {
			j, v, err := ler(rows)
			if err != nil {
				rows.Close()
				return err
			}
			lote = append(lote, travado{j, v})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, t := range lote {
			evs, err := decidir(t.j)
			if err != nil {
				return err
			}
			j, err := job.ApplyAll(t.j, evs)
			if err != nil {
				return err
			}
			if err := gravar(ctx, tx, j, evs, t.v); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	for _, p := range perdidas {
		s.dono.Perdeu(p)
	}
	return n, err
}

// Esperando conta os jobs do namespace que esperam execução: prontos
// ou aguardando o retry. Agendados para o futuro não contam — não são
// atraso, são compromisso. Usa o índice job_consulta.
func (s *Store) Esperando(ctx context.Context, ns string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM job
		WHERE namespace = $1 AND state IN ('available', 'retryable')`,
		ns).Scan(&n)
	return n, err
}
