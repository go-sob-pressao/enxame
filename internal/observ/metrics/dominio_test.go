package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestTentativasPorResultado(t *testing.T) {
	inicio := time.Now()
	JobTerminou("teste", "concluido", inicio, inicio.Add(time.Second))
	JobTerminou("teste", "retry", inicio, inicio.Add(time.Second))
	JobTerminou("teste", "concluido", inicio, inicio.Add(time.Second))
	if v := testutil.ToFloat64(tentativas.WithLabelValues("teste",
		"concluido")); v != 2 {
		t.Fatalf("concluídos: %v", v)
	}
	JobReservado("teste", inicio, inicio.Add(-time.Second)) // relógio
	n := testutil.CollectAndCount(atraso, "enxame_job_atraso_segundos")
	if n != 1 {
		t.Fatalf("%d séries de atraso", n)
	}
}
