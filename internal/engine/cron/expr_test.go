package cron_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/engine/cron"
)

func TestNext(t *testing.T) {
	sp, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 9, 25, 10, 2, 30, 0, time.UTC) // sexta
	casos := []struct {
		expr     string
		loc      *time.Location
		esperado string
	}{
		{"*/5 * * * *", time.UTC, "2026-09-25T10:05:00Z"},
		{"0 * * * *", time.UTC, "2026-09-25T11:00:00Z"},
		{"30 9 * * 1-5", time.UTC, "2026-09-28T09:30:00Z"},
		{"0 0 1 * *", time.UTC, "2026-10-01T00:00:00Z"},
		{"0 0 29 2 *", time.UTC, "2028-02-29T00:00:00Z"},
		// 10h02 em UTC são 7h02 em São Paulo.
		{"0 8 * * *", sp, "2026-09-25T08:00:00-03:00"},
		// Dia 13 ou sexta-feira: a próxima sexta vem antes do dia 13.
		{"0 0 13 * 5", time.UTC, "2026-10-02T00:00:00Z"},
	}
	for _, c := range casos {
		e, err := cron.Parse(c.expr)
		if err != nil {
			t.Fatal(err)
		}
		got := e.Next(base, c.loc).Format(time.RFC3339)
		if got != c.esperado {
			t.Errorf("%q: %s; esperado %s", c.expr, got, c.esperado)
		}
	}
	for _, ruim := range []string{"* * * *", "61 * * * *", "*/0 * * * *",
		"5-1 * * * *", "a * * * *"} {
		if _, err := cron.Parse(ruim); err == nil {
			t.Errorf("%q aceita", ruim)
		}
	}
}
