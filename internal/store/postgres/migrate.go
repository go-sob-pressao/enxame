package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var migracoes embed.FS

// Migrate aplica todas as migrações up, em ordem de nome, num banco
// vazio. Controle de versão de esquema entra no Capítulo 17.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	nomes, err := fs.Glob(migracoes, "migrations/*.up.sql")
	if err != nil {
		return err
	}
	slices.Sort(nomes)
	for _, n := range nomes {
		sql, err := migracoes.ReadFile(n)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf(
				"migração %s: %w",
				strings.TrimPrefix(n, "migrations/"),
				err,
			)
		}
	}
	return nil
}
