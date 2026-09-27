package partition

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Lease disputa e mantém a posse de partições em partition_lease.
// Todos os instantes são do relógio do banco.
type Lease struct {
	DB      *pgxpool.Pool
	No      string        // quem pede
	Duracao time.Duration // quanto a posse dura sem renovação
}

// livro:inicio aquisicao

// Adquirir toma a partição p se ela estiver livre ou vencida, ou se já
// for deste nó, e incrementa o range_id: cada posse nova tem um token
// maior que todas as anteriores. Devolve o token e se conseguiu.
func (l Lease) Adquirir(
	ctx context.Context,
	p int,
) (int64, bool, error) {
	var token int64
	err := l.DB.QueryRow(ctx, `UPDATE partition_lease
		SET owner = $1, range_id = range_id + 1,
		    lease_expires_at = now() + $2::interval
		WHERE partition_id = $3
		  AND (owner IS NULL OR owner = $1 OR lease_expires_at < now())
		RETURNING range_id`, l.No, l.Duracao.String(), p).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil // outro nó tem a posse, e ela vale
	}
	return token, err == nil, err
}

// Renovar estende a posse, se o token ainda for o vigente. Renovar não
// muda o token: é a mesma posse, por mais tempo.
func (l Lease) Renovar(ctx context.Context, p int, token int64) (bool,
	error) {
	tag, err := l.DB.Exec(ctx, `UPDATE partition_lease
		SET lease_expires_at = now() + $1::interval
		WHERE partition_id = $2 AND owner = $3 AND range_id = $4`,
		l.Duracao.String(), p, l.No, token)
	return tag.RowsAffected() == 1, err
}

// livro:fim aquisicao

// Soltar devolve a partição antes do vencimento, no desligamento: o
// próximo dono não precisa esperar o lease vencer.
func (l Lease) Soltar(ctx context.Context, p int, token int64) error {
	_, err := l.DB.Exec(ctx, `UPDATE partition_lease
		SET owner = NULL, lease_expires_at = NULL
		WHERE partition_id = $1 AND owner = $2 AND range_id = $3`,
		p, l.No, token)
	return err
}

// AdquirirVarias tenta adquirir cada partição de ps, numa instrução só,
// e devolve as que conseguiu, com o token de cada uma.
func (l Lease) AdquirirVarias(
	ctx context.Context,
	ps []int,
) (map[int]int64, error) {
	// As linhas são travadas em ordem de partição, pulando as que outra
	// transação segura: duas aquisições, ou uma aquisição e a cerca de
	// um dono, nunca esperam uma pela outra em ordens opostas.
	rows, err := l.DB.Query(ctx, `WITH alvo AS (
		  SELECT partition_id FROM partition_lease
		   WHERE partition_id = ANY($3)
		     AND (owner IS NULL OR owner = $1
		          OR lease_expires_at < now())
		   ORDER BY partition_id FOR UPDATE SKIP LOCKED)
		UPDATE partition_lease l
		   SET owner = $1, range_id = range_id + 1,
		       lease_expires_at = now() + $2::interval
		  FROM alvo WHERE l.partition_id = alvo.partition_id
		RETURNING l.partition_id, l.range_id`,
		l.No, l.Duracao.String(), int32s(ps))
	if err != nil {
		return nil, err
	}
	return tokens(rows)
}

// RenovarVarias renova as posses dadas, numa instrução só, e devolve as
// que continuam deste nó; as que faltarem foram perdidas.
func (l Lease) RenovarVarias(
	ctx context.Context,
	posses map[int]int64,
) (map[int]int64, error) {
	var ps []int32
	var rs []int64
	for p, r := range posses {
		ps = append(ps, int32(p)) //nolint:gosec // < 512
		rs = append(rs, r)
	}
	rows, err := l.DB.Query(ctx, `WITH alvo AS (
		  SELECT l.partition_id FROM partition_lease l
		    JOIN unnest($2::int[], $3::bigint[]) AS t(p, r)
		      ON l.partition_id = t.p AND l.range_id = t.r
		   WHERE l.owner = $4
		   ORDER BY l.partition_id FOR UPDATE)
		UPDATE partition_lease l
		   SET lease_expires_at = now() + $1::interval
		  FROM alvo WHERE l.partition_id = alvo.partition_id
		RETURNING l.partition_id, l.range_id`,
		l.Duracao.String(), ps, rs, l.No)
	if err != nil {
		return nil, err
	}
	return tokens(rows)
}

func tokens(rows pgx.Rows) (map[int]int64, error) {
	defer rows.Close()
	t := map[int]int64{}
	for rows.Next() {
		var p int
		var r int64
		if err := rows.Scan(&p, &r); err != nil {
			return nil, err
		}
		t[p] = r
	}
	return t, rows.Err()
}

func int32s(ps []int) []int32 {
	r := make([]int32, len(ps))
	for i, p := range ps {
		r[i] = int32(p) //nolint:gosec // < 512
	}
	return r
}
