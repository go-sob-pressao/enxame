package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// sessoes prepara o banco — partição 0 com range_id 5, do nó a — e
// abre duas conexões: a do zumbi (a) e a do novo dono (b).
func sessoes(t *testing.T) (*pgxpool.Pool, *pgx.Conn, *pgx.Conn) {
	t.Helper()
	db := testutil.Postgres(t)
	ctx := t.Context()
	for _, sql := range []string{
		`UPDATE partition_lease SET range_id = 5, owner = 'a'
		 WHERE partition_id = 0`,
		`CREATE TABLE extrato (id SERIAL, dono TEXT, token BIGINT)`,
	} {
		if _, err := db.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	conectar := func() *pgx.Conn {
		c, err := db.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(c.Release)
		return c.Conn()
	}
	return db, conectar(), conectar()
}

// aquisicao é o UPDATE do novo dono, numa goroutine: ele pode ficar
// esperando um lock, e o teste precisa ver isso acontecer.
func aquisicao(ctx context.Context, b *pgx.Conn) <-chan error {
	fim := make(chan error, 1)
	go func() {
		_, err := b.Exec(ctx, `UPDATE partition_lease
			SET owner = 'b', range_id = range_id + 1
			WHERE partition_id = 0`)
		fim <- err
	}()
	return fim
}

// esperandoLock diz se a sessão b está parada esperando um lock,
// pelo pg_stat_activity — sem dormir para ver.
func esperandoLock(t *testing.T, db *pgxpool.Pool, b *pgx.Conn) bool {
	t.Helper()
	for range 200 {
		var espera string
		_ = db.QueryRow(t.Context(), `SELECT coalesce(wait_event_type,
			'') FROM pg_stat_activity WHERE pid = $1`,
			b.PgConn().PID()).Scan(&espera)
		if espera == "Lock" {
			return true
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-t.Context().Done():
			return false
		}
	}
	return false
}

// livro:inicio subconsulta

// A verificação por subconsulta: o zumbi confere o token dentro do
// próprio INSERT. Em READ COMMITTED, a subconsulta lê o que estava
// comitado quando a instrução começou, e não trava nada.
func TestFencingPorSubconsultaDeixaOZumbiEscrever(t *testing.T) {
	db, a, b := sessoes(t)
	ctx := t.Context()
	tx, _ := a.Begin(ctx)
	_, err := tx.Exec(ctx, `INSERT INTO extrato (dono, token)
		SELECT 'a', 5 WHERE 5 = (SELECT range_id FROM partition_lease
		                         WHERE partition_id = 0)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("a: INSERT com o token 5 conferido na subconsulta")
	if err := <-aquisicao(ctx, b); err != nil { // não espera ninguém
		t.Fatal(err)
	}
	t.Log("b: aquisição comitada; range_id = 6")
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Log("a: COMMIT aceito — depois da aquisição de b")
	var n int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM extrato
		WHERE token = 5`).Scan(&n)
	if n != 1 {
		t.Fatalf("esperava a escrita do zumbi no extrato; há %d", n)
	}
}

// livro:fim subconsulta

// livro:inicio for-share

// A verificação com FOR SHARE, como em ConferirCerca: a aquisição
// espera o zumbi terminar, e tudo o que ele escreveu fica antes da
// troca de dono. A transação seguinte dele é recusada.
func TestFencingComForShareBarraOZumbi(t *testing.T) {
	db, a, b := sessoes(t)
	ctx := t.Context()
	cerca := postgres.Cerca{Particao: 0, RangeID: 5}
	tx, _ := a.Begin(ctx)
	if err := postgres.ConferirCerca(ctx, tx, cerca); err != nil {
		t.Fatal(err)
	}
	_, _ = tx.Exec(ctx, `INSERT INTO extrato (dono, token) VALUES
		('a', 5)`)
	t.Log("a: FOR SHARE conferiu o token 5; INSERT")
	fim := aquisicao(ctx, b)
	if !esperandoLock(t, db, b) {
		t.Fatal("a aquisição não esperou o zumbi")
	}
	t.Log("b: aquisição esperando o lock de a")
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-fim; err != nil {
		t.Fatal(err)
	}
	t.Log("a: COMMIT; só então b adquire: range_id = 6")
	tx, _ = a.Begin(ctx)
	err := postgres.ConferirCerca(ctx, tx, cerca)
	_ = tx.Rollback(ctx)
	if !errors.Is(err, store.ErrCercado) {
		t.Fatalf("a escreveu de novo com o token 5: %v", err)
	}
	t.Logf("a: próxima transação recusada: %v", err)
}

// livro:fim for-share
