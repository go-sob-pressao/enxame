// Package lease é o enigma do Capítulo 22: o lease que expirava antes
// de começar.
package lease

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Criar cria a tabela do lease, com uma linha livre.
func Criar(ctx context.Context, db *pgxpool.Pool, nome string) error {
	_, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS lease (
		nome   TEXT PRIMARY KEY,
		dono   TEXT NOT NULL DEFAULT '',
		expira TIMESTAMPTZ NOT NULL DEFAULT '-infinity')`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO lease (nome) VALUES ($1)
		ON CONFLICT DO NOTHING`, nome)
	return err
}

// livro:inicio lease-errado

// No é um nó que disputa o lease com o relógio que tem.
type No struct {
	Nome    string
	Relogio func() time.Time // o relógio de parede desta máquina
	DB      *pgxpool.Pool
}

// Adquirir toma o lease se ele for deste nó ou estiver vencido, e o
// renova por d. O vencimento é calculado e comparado com o relógio do
// nó.
func (n No) Adquirir(ctx context.Context, nome string,
	d time.Duration) (bool, error) {
	agora := n.Relogio()
	tag, err := n.DB.Exec(ctx, `UPDATE lease SET dono = $1, expira = $2
		WHERE nome = $3 AND (dono = $1 OR expira < $4)`,
		n.Nome, agora.Add(d), nome, agora)
	return tag.RowsAffected() == 1, err
}

// livro:fim lease-errado

// livro:inicio lease-banco

// AdquirirPeloBanco faz o mesmo com um relógio só, o do banco: o
// vencimento é calculado e comparado com now(), e o relógio do nó não
// entra na decisão.
func (n No) AdquirirPeloBanco(ctx context.Context, nome string,
	d time.Duration) (bool, error) {
	tag, err := n.DB.Exec(ctx, `UPDATE lease
		SET dono = $1, expira = now() + $2::interval
		WHERE nome = $3 AND (dono = $1 OR expira < now())`,
		n.Nome, d.String(), nome)
	return tag.RowsAffected() == 1, err
}

// livro:fim lease-banco
