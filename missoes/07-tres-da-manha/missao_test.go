package tresdamanha_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/go-sob-pressao/enxame/internal/observ/logging"
	"github.com/go-sob-pressao/enxame/internal/observ/metrics"
	tresdamanha "github.com/go-sob-pressao/enxame/missoes/07-tres-da-manha"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

type fechamento struct {
	Cliente string `json:"cliente"`
}

func (fechamento) Kind() string { return "fechamento" }

// livro:inicio missao-07-teste

// A noite do chamado, em 25 segundos: 20 relatórios de fechamento
// enfileirados às 3h, o worker do deploy das 2h50 rodando. Todos têm de
// ficar prontos. Seja qual for o resultado, a telemetria da noite fica
// em telemetria/: as métricas, os spans e os logs.
func TestMissao(t *testing.T) {
	gravador := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(gravador)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	var logs bytes.Buffer
	log := slog.New(logging.Correlacao{
		Handler: slog.NewJSONHandler(&logs, nil)})

	db := testutil.Postgres(t)
	c := enxame.New(db, "loja")
	for i := range 20 {
		ctx, span := otel.Tracer("agenda").Start(t.Context(),
			"fechamento do mês")
		_, err := c.Insert(ctx, fechamento{fmt.Sprintf("cliente-%02d", i)},
			enxame.Queue("relatorios"))
		span.End()
		if err != nil {
			t.Fatal(err)
		}
	}
	w := c.NewWorker(tresdamanha.Config(log))
	w.Handle("fechamento", tresdamanha.Relatorio(log, 3*time.Second))
	ctx, parar := context.WithTimeout(t.Context(), 25*time.Second)
	defer parar()
	fim := make(chan error, 1)
	go func() { fim <- w.Run(ctx) }()
	prontos := 0
	for ctx.Err() == nil && prontos < 20 {
		<-time.After(500 * time.Millisecond)
		_ = db.QueryRow(t.Context(), `SELECT count(*) FROM job
			WHERE queue = 'relatorios' AND state = 'completed'`).
			Scan(&prontos)
	}
	parar()
	<-fim
	gravarTelemetria(t, gravador, logs.Bytes())
	t.Logf("%d de 20 relatórios prontos; telemetria em telemetria/",
		prontos)
	if prontos < 20 {
		t.Fatal("a noite acabou com relatórios faltando")
	}
}

// livro:fim missao-07-teste

// gravarTelemetria escreve o que a noite deixou: métricas no formato do
// Prometheus, um span por linha e as linhas de log.
func gravarTelemetria(t *testing.T, g *tracetest.SpanRecorder,
	logs []byte) {
	dir := "telemetria"
	_ = os.MkdirAll(dir, 0o755)
	var met bytes.Buffer
	familias, err := metrics.Registro.Gather()
	if err == nil {
		enc := expfmt.NewEncoder(&met,
			expfmt.NewFormat(expfmt.TypeTextPlain))
		for _, f := range familias {
			if f.GetName()[:7] == "enxame_" {
				_ = enc.Encode(f)
			}
		}
	}
	var spans bytes.Buffer
	for _, s := range g.Ended() {
		atr := map[string]string{}
		for _, a := range s.Attributes() {
			atr[string(a.Key)] = a.Value.String()
		}
		linha, _ := json.Marshal(map[string]any{
			"nome":      s.Name(),
			"trace_id":  s.SpanContext().TraceID().String(),
			"span_id":   s.SpanContext().SpanID().String(),
			"inicio":    s.StartTime().Format(time.RFC3339Nano),
			"duracao":   s.EndTime().Sub(s.StartTime()).String(),
			"status":    s.Status().Code.String(),
			"erro":      s.Status().Description,
			"atributos": atr,
		})
		spans.Write(append(linha, '\n'))
	}
	for nome, b := range map[string][]byte{"metricas.prom": met.Bytes(),
		"spans.jsonl": spans.Bytes(), "logs.jsonl": logs} {
		if err := os.WriteFile(filepath.Join(dir, nome), b,
			0o644); err != nil {
			t.Error(err)
		}
	}
}
