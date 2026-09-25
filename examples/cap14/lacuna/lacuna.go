// Package lacuna é o enigma do Capítulo 14: um leitor de eventos que
// anda por um id tirado de uma sequence, e perde o evento cuja
// transação fez COMMIT depois de outra com id maior.
package lacuna

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Esquema cria as tabelas do exemplo.
const Esquema = `
CREATE TABLE evento_global (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dado TEXT NOT NULL
);
CREATE TABLE fluxo (
    id       INT    PRIMARY KEY,
    next_seq BIGINT NOT NULL DEFAULT 1
);
CREATE TABLE evento_fluxo (
    fluxo INT    NOT NULL REFERENCES fluxo (id),
    seq   BIGINT NOT NULL,
    dado  TEXT   NOT NULL,
    PRIMARY KEY (fluxo, seq)
);`

// livro:inicio lacuna-defeito

// Gravar acrescenta um evento; o id vem da sequence da coluna identity,
// alocado no INSERT — e não no COMMIT.
func Gravar(ctx context.Context, tx pgx.Tx, dado string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO evento_global (dado) VALUES ($1)`, dado)
	return err
}

// Leitor entrega cada evento uma vez, lembrando o maior id já lido.
type Leitor struct{ ultimo int64 }

// Ler devolve os eventos novos desde a última leitura.
func (l *Leitor) Ler(ctx context.Context, db *pgxpool.Pool) (
	[]string, error,
) {
	rows, err := db.Query(ctx, `SELECT id, dado FROM evento_global
		WHERE id > $1 ORDER BY id`, l.ultimo)
	if err != nil {
		return nil, err
	}
	var novos []string
	for rows.Next() {
		var dado string
		if err := rows.Scan(&l.ultimo, &dado); err != nil {
			return nil, err
		}
		novos = append(novos, dado)
	}
	return novos, rows.Err()
}

// livro:fim lacuna-defeito

// livro:inicio lacuna-correto

// GravarNoFluxo acrescenta um evento ao fluxo com o seq tirado de
// fluxo.next_seq. O UPDATE trava a linha do fluxo até o COMMIT: a
// próxima transação que gravar no mesmo fluxo espera, e o seq segue a
// ordem de COMMIT.
func GravarNoFluxo(
	ctx context.Context,
	tx pgx.Tx,
	fluxo int,
	dado string,
) error {
	_, err := tx.Exec(ctx, `WITH s AS (
			UPDATE fluxo SET next_seq = next_seq + 1 WHERE id = $1
			RETURNING next_seq - 1 AS seq)
		INSERT INTO evento_fluxo (fluxo, seq, dado)
		SELECT $1, seq, $2 FROM s`, fluxo, dado)
	return err
}

// livro:fim lacuna-correto

// LeitorDoFluxo entrega cada evento de um fluxo uma vez, pelo seq.
type LeitorDoFluxo struct {
	Fluxo  int
	ultimo int64
}

// Ler devolve os eventos novos do fluxo desde a última leitura.
func (l *LeitorDoFluxo) Ler(ctx context.Context, db *pgxpool.Pool) (
	[]string, error,
) {
	rows, err := db.Query(ctx, `SELECT seq, dado FROM evento_fluxo
		WHERE fluxo = $1 AND seq > $2 ORDER BY seq`, l.Fluxo, l.ultimo)
	if err != nil {
		return nil, err
	}
	var novos []string
	for rows.Next() {
		var dado string
		if err := rows.Scan(&l.ultimo, &dado); err != nil {
			return nil, err
		}
		novos = append(novos, dado)
	}
	return novos, rows.Err()
}
