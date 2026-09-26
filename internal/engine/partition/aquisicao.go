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
