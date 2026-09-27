//go:build integration

package integration_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio relogio-torto

// A regressão do Experimento 3 do Capítulo 28: um nó com o relógio
// torto — adiantado ou atrasado — não decide quais janelas vencem. O
// Store com o relógio do banco dispara a janela vencida no now() dele,
// não no instante que o nó passa.
func TestDisparoComRelogioTorto(t *testing.T) {
	for _, desvio := range []time.Duration{3 * time.Minute,
		-3 * time.Minute} {
		t.Run(desvio.String(), func(t *testing.T) {
			db := testutil.Postgres(t)
			s := postgres.New(db).RelogioDoBanco()
			var agora time.Time
			_ = db.QueryRow(t.Context(), `SELECT now()`).Scan(&agora)
			// Uma janela que ainda não venceu, e uma que já venceu.
			for id, janela := range map[string]time.Time{
				"futura":  agora.Add(time.Minute),
				"vencida": agora.Add(-time.Second)} {
				if err := s.UpsertSchedule(t.Context(),
					schedule.Schedule{Namespace: "loja", ID: id,
						Expr: "* * * * *", Timezone: "UTC",
						Queue: "q", Kind: "k",
						NextFire: janela}); err != nil {
					t.Fatal(err)
				}
			}
			d, _, err := s.FireDue(t.Context(),
				time.Now().Add(desvio), schedule.Disparo)
			if err != nil || d != 1 {
				t.Fatalf("relógio %+v: %d disparos (%v); esperava "+
					"só a janela vencida", desvio, d, err)
			}
		})
	}
}

// livro:fim relogio-torto
