// Package expand mostra uma mudança de coluna em três deploys: o valor
// do pedido sai de NUMERIC em reais para BIGINT em centavos.
package expand

import (
	"context"
	"embed"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

//go:embed migracoes/*.sql
var arquivos embed.FS

// Migrar leva o esquema do exemplo à versão alvo.
func Migrar(ctx context.Context, db *pgxpool.Pool, alvo int) error {
	sub, err := fs.Sub(arquivos, "migracoes")
	if err != nil {
		return err
	}
	ms, err := postgres.Carregar(sub)
	if err != nil {
		return err
	}
	return postgres.Migrar(ctx, db, "exemplo_schema", ms, alvo)
}

// livro:inicio versoes

// V1 é o código de antes: só conhece valor, em reais.
type V1 struct{ DB *pgxpool.Pool }

// Gravar grava um pedido.
func (v V1) Gravar(ctx context.Context, id int, reais float64) error {
	_, err := v.DB.Exec(ctx,
		`INSERT INTO pedido (id, valor) VALUES ($1, $2)`, id, reais)
	return err
}

// V2 é o código de transição: escreve as duas colunas e lê a nova.
type V2 struct{ DB *pgxpool.Pool }

// Gravar grava um pedido.
func (v V2) Gravar(ctx context.Context, id int, centavos int64) error {
	_, err := v.DB.Exec(ctx, `INSERT INTO pedido (id, valor,
		valor_centavos) VALUES ($1, $2::bigint / 100.0, $2)`,
		id, centavos)
	return err
}

// Total soma os pedidos, em centavos.
func (v V2) Total(ctx context.Context) (int64, error) {
	var t int64
	err := v.DB.QueryRow(ctx,
		`SELECT coalesce(sum(valor_centavos), 0) FROM pedido`).Scan(&t)
	return t, err
}

// V3 é o código de depois: só conhece valor_centavos.
type V3 struct{ DB *pgxpool.Pool }

// Gravar grava um pedido.
func (v V3) Gravar(ctx context.Context, id int, centavos int64) error {
	_, err := v.DB.Exec(ctx, `INSERT INTO pedido (id, valor_centavos)
		VALUES ($1, $2)`, id, centavos)
	return err
}

// livro:fim versoes

// Backfill preenche valor_centavos nas linhas antigas, em lotes.
func Backfill(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	return postgres.EmLotes(ctx, db, `UPDATE pedido
		SET valor_centavos = round(valor * 100)
		WHERE id IN (SELECT id FROM pedido
		             WHERE valor_centavos IS NULL
		             LIMIT $1 FOR UPDATE SKIP LOCKED)`, 100, 0)
}
