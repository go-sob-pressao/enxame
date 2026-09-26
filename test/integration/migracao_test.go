//go:build integration

package integration_test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// esquema resume o esquema público — tabelas, colunas, restrições e
// índices — numa impressão digital.
func esquema(t *testing.T, db *pgxpool.Pool) string {
	t.Helper()
	rows, err := db.Query(t.Context(), `
		SELECT 'c ' || table_name || '.' || column_name || ' ' ||
		       data_type || ' ' || is_nullable || ' ' ||
		       coalesce(column_default, '')
		  FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name <> 'enxame_schema'
		UNION ALL
		SELECT 'r ' || conrelid::regclass || ' ' || conname || ' ' ||
		       pg_get_constraintdef(oid)
		  FROM pg_constraint WHERE connamespace = 'public'::regnamespace
		UNION ALL
		SELECT 'i ' || indexdef FROM pg_indexes
		 WHERE schemaname = 'public'
		   AND tablename <> 'enxame_schema'
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var linhas []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			t.Fatal(err)
		}
		linhas = append(linhas, l)
	}
	h := sha256.Sum256([]byte(strings.Join(linhas, "\n")))
	return fmt.Sprintf("%d objetos, %x", len(linhas), h[:6])
}

func tabelas(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := db.QueryRow(t.Context(), `SELECT count(*)
		FROM information_schema.tables WHERE table_schema = 'public'
		  AND table_name <> 'enxame_schema'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// livro:inicio ida-e-volta

// Experimento 17.1: todas as migrações sobem, descem até o zero e sobem
// de novo; o esquema final é idêntico ao primeiro.
func TestMigracoesIdaEVolta(t *testing.T) {
	db := testutil.Postgres(t) // já migrado até a última versão
	ms, err := postgres.Migracoes()
	if err != nil {
		t.Fatal(err)
	}
	antes := esquema(t, db)
	if err := postgres.Migrar(t.Context(), db, "enxame_schema", ms,
		0); err != nil {
		t.Fatal(err)
	}
	if n := tabelas(t, db); n != 0 {
		t.Fatalf("%d tabelas depois de descer tudo", n)
	}
	if err := postgres.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	depois := esquema(t, db)
	t.Logf("esquema: %s", depois)
	if antes != depois {
		t.Fatalf("antes %s; depois %s", antes, depois)
	}
}

// livro:fim ida-e-volta

// A volta da 0007 recusa quando já existe um marcador de versão: ela
// não apaga histórico para caber no esquema antigo.
func TestVoltaRecusaPerderHistorico(t *testing.T) {
	db := testutil.Postgres(t)
	ctx := t.Context()
	if _, err := db.Exec(ctx, `
		INSERT INTO workflow_run (run_id, partition_id, namespace,
		  workflow_id, workflow_type, code_version, queue)
		VALUES ('0192a3b4-0000-7000-8000-000000000001', 0, 'ns', 'w',
		  'f', 1, 'workflow');
		INSERT INTO workflow_step (run_id, step_seq, step_name,
		  step_kind, state)
		VALUES ('0192a3b4-0000-7000-8000-000000000001', 1, 'nota-v2',
		  'version', 'completed')`); err != nil {
		t.Fatal(err)
	}
	ms, _ := postgres.Migracoes()
	err := postgres.Migrar(ctx, db, "enxame_schema", ms, 6)
	if err == nil {
		t.Fatal("a volta da 0007 apagaria o marcador; deveria recusar")
	}
	t.Log(err)
	var v int
	if err := db.QueryRow(ctx,
		`SELECT max(versao) FROM enxame_schema`).Scan(&v); err != nil || v != 7 {
		t.Fatalf("versão %d, %v", v, err)
	}
}
