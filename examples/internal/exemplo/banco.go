// Package exemplo tem o que os exemplos numerados compartilham: um
// banco próprio, recriado a cada execução, com o esquema do Enxame.
package exemplo

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// Banco recria o banco nome no servidor de ENXAME_DB_DSN e o migra.
func Banco(ctx context.Context, nome string) (*pgxpool.Pool, error) {
	dsn := os.Getenv("ENXAME_DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("defina ENXAME_DB_DSN (make up)")
	}
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer func() { _ = admin.Close(ctx) }()
	for _, sql := range []string{
		"DROP DATABASE IF EXISTS " + nome + " WITH (FORCE)",
		"CREATE DATABASE " + nome,
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			return nil, err
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Database = nome
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return db, enxame.Migrate(ctx, db)
}

// Esperar chama cancel quando não houver mais jobs pendentes no banco.
func Esperar(ctx context.Context, db *pgxpool.Pool, cancel func()) {
	for ctx.Err() == nil {
		var n int
		_ = db.QueryRow(ctx, `SELECT count(*) FROM job
			WHERE state NOT IN ('completed', 'discarded')`).Scan(&n)
		if n == 0 {
			cancel()
			return
		}
		<-time.After(100 * time.Millisecond)
	}
}
