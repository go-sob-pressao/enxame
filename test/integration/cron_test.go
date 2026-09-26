//go:build integration

package integration_test

import (
	"sync"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	"github.com/go-sob-pressao/enxame/internal/engine/cron"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

var _ cron.Store = (*postgres.Store)(nil)

func agendamento(t *testing.T, s *postgres.Store, prox time.Time) {
	t.Helper()
	if err := s.UpsertSchedule(t.Context(), schedule.Schedule{
		Namespace: "loja", ID: "fechamento", Expr: "*/5 * * * *",
		Timezone: "UTC", Queue: "q", Kind: "fechar-caixa",
		NextFire: prox,
	}); err != nil {
		t.Fatal(err)
	}
}

// livro:inicio cron-teste

// A janela das 10h00 é disparada; depois, o agendamento é regravado
// com a mesma janela — um operador que salva de novo a configuração,
// um líder antigo que ainda não sabe que perdeu o posto. O segundo
// disparo é absorvido pela chave da janela.
func TestCronUmaVezPorJanela(t *testing.T) {
	db := testutil.Postgres(t)
	s := postgres.New(db)
	dez := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	agendamento(t, s, dez)
	agora := dez.Add(30 * time.Second)
	d, a, err := s.FireDue(t.Context(), agora, schedule.Disparo)
	if err != nil || d != 1 || a != 0 {
		t.Fatalf("primeiro disparo: %d, %d, %v", d, a, err)
	}
	agendamento(t, s, dez) // a mesma janela, de novo
	d, a, err = s.FireDue(t.Context(), agora, schedule.Disparo)
	if err != nil || d != 0 || a != 1 {
		t.Fatalf("segundo disparo: %d, %d, %v", d, a, err)
	}
	var jobs int
	var chave string
	if err := db.QueryRow(t.Context(), `SELECT count(*), max(unique_key)
		FROM job WHERE kind = 'fechar-caixa'`).Scan(&jobs,
		&chave); err != nil {
		t.Fatal(err)
	}
	esperada := "cron:loja:fechamento:2026-09-25T10:00:00Z"
	if jobs != 1 || chave != esperada {
		t.Fatalf("%d jobs, chave %s", jobs, chave)
	}
}

// livro:fim cron-teste

// Dois agendadores ao mesmo tempo: SKIP LOCKED dá a janela a um só.
func TestCronDoisAgendadores(t *testing.T) {
	db := testutil.Postgres(t)
	s := postgres.New(db)
	dez := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	agendamento(t, s, dez)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, _, err := s.FireDue(t.Context(),
				dez.Add(time.Second), schedule.Disparo); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	var jobs int
	var prox time.Time
	if err := db.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM job), next_fire_at FROM schedule`).Scan(
		&jobs, &prox); err != nil {
		t.Fatal(err)
	}
	if jobs != 1 || !prox.Equal(dez.Add(5*time.Minute)) {
		t.Fatalf("%d jobs, próxima %v", jobs, prox)
	}
}
