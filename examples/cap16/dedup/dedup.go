// Package dedup é a tabela de deduplicação própria, para quando o
// sistema externo não aceita chave de idempotência.
package dedup

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Esquema cria a tabela de efeitos.
const Esquema = `CREATE TABLE efeito (
    chave  TEXT        PRIMARY KEY,
    estado TEXT        NOT NULL
           CHECK (estado IN ('iniciado', 'concluido')),
    em     TIMESTAMPTZ NOT NULL DEFAULT now()
);`

// ErrIncerto indica que outra tentativa começou o efeito e não
// registrou o fim. Ele pode ter acontecido ou não; só quem conhece o
// efeito sabe o que fazer.
var ErrIncerto = errors.New("efeito iniciado sem conclusão registrada")

// Dedup executa efeitos uma vez por chave. Falha, quando não é nil, é
// chamada entre o efeito e o registro da conclusão: é onde o teste
// injeta a queda do processo.
type Dedup struct {
	DB    *pgxpool.Pool
	Falha func() error
}

// livro:inicio dedup

// Executar faz o efeito se a chave nunca foi vista. Três passos, em
// três transações: reservar a chave, fazer o efeito, marcar concluído.
// Uma queda entre o efeito e a marca deixa a chave em "iniciado", e a
// próxima tentativa recebe ErrIncerto em vez de repetir às cegas.
func (d *Dedup) Executar(
	ctx context.Context,
	chave string,
	efeito func(context.Context) error,
) error {
	tag, err := d.DB.Exec(ctx, `INSERT INTO efeito (chave, estado)
		VALUES ($1, 'iniciado') ON CONFLICT (chave) DO NOTHING`, chave)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 { // a chave já existia
		var estado string
		err := d.DB.QueryRow(ctx, `SELECT estado FROM efeito
			WHERE chave = $1`, chave).Scan(&estado)
		if err != nil || estado == "concluido" {
			return err
		}
		return ErrIncerto
	}
	if err := efeito(ctx); err != nil {
		// Falha definitiva do efeito: libera a chave para outra vez.
		_, _ = d.DB.Exec(ctx, `DELETE FROM efeito
			WHERE chave = $1 AND estado = 'iniciado'`, chave)
		return err
	}
	if d.Falha != nil {
		if err := d.Falha(); err != nil {
			return err
		}
	}
	_, err = d.DB.Exec(ctx, `UPDATE efeito SET estado = 'concluido'
		WHERE chave = $1`, chave)
	return err
}

// livro:fim dedup

// livro:inicio dedup-local

// ExecutarLocal é para efeitos no mesmo banco: a chave e o efeito
// entram na mesma transação, e a janela desaparece. Exatamente uma vez,
// de verdade — mas só para o que mora no banco.
func (d *Dedup) ExecutarLocal(
	ctx context.Context,
	chave string,
	efeito func(context.Context, pgx.Tx) error,
) error {
	return pgx.BeginFunc(ctx, d.DB, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO efeito (chave, estado)
			VALUES ($1, 'concluido') ON CONFLICT (chave) DO NOTHING`,
			chave)
		if err != nil || tag.RowsAffected() == 0 {
			return err // já feito: nada a fazer
		}
		return efeito(ctx, tx)
	})
}

// livro:fim dedup-local
