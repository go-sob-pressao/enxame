package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

// livro:inicio postgres-efemero

// Postgres devolve um pool conectado a um banco criado só para este
// teste, com as migrações aplicadas, e apagado quando o teste termina.
// O servidor vem de ENXAME_DB_DSN (make up, ou o serviço da CI); sem a
// variável, o teste é pulado, e não falha.
func Postgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("ENXAME_DB_DSN")
	if dsn == "" {
		t.Skip("ENXAME_DB_DSN não definido: rode make up")
	}
	ctx := context.Background() // a limpeza roda depois de t.Context
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	nome := "enxame_t_" + aleatorio()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+nome); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = nome
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+nome+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// livro:fim postgres-efemero

func aleatorio() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
