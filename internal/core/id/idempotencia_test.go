package id_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
)

func TestChavesEstaveis(t *testing.T) {
	jid, err := id.ParseJobID("0192a3b4-0000-7000-8000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	sp := time.FixedZone("BRT", -3*3600)
	casos := []struct{ obtida, esperada string }{
		{id.JobKey("loja", jid),
			"job:loja:0192a3b4-0000-7000-8000-000000000001"},
		{id.StepKey("r1", 3), "step:r1:3"},
		// O mesmo instante em fusos diferentes é a mesma janela.
		{id.WindowKey("loja", "fechamento",
			time.Date(2026, 9, 25, 0, 0, 0, 0, sp)),
			"cron:loja:fechamento:2026-09-25T03:00:00Z"},
	}
	for _, c := range casos {
		if c.obtida != c.esperada {
			t.Errorf("%q; esperada %q", c.obtida, c.esperada)
		}
	}
}
