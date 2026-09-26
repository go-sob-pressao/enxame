package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// livro:inicio lote

// EmLotes repete sql — uma instrução que processa até $1 linhas ainda
// não processadas — até que um lote não processe nenhuma. Cada lote é
// uma transação curta: trava poucas linhas por pouco tempo, e o sistema
// continua atendendo entre um lote e outro. A pausa devolve fôlego ao
// banco; o contexto interrompe a migração, que pode ser retomada de
// onde parou, porque o sql só escolhe linhas ainda não processadas.
func EmLotes(
	ctx context.Context,
	db *pgxpool.Pool,
	sql string,
	tamanho int,
	pausa time.Duration,
) (int64, error) {
	var total int64
	for {
		tag, err := db.Exec(ctx, sql, tamanho)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
		if tag.RowsAffected() == 0 {
			return total, nil
		}
		select {
		case <-time.After(pausa):
		case <-ctx.Done():
			return total, ctx.Err()
		}
	}
}

// livro:fim lote
