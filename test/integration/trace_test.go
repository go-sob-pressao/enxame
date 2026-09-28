//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio teste-trace

// O trace atravessa o banco. Um job criado por uma requisição com
// traceparent roda, noutro momento e noutra goroutine, como um span do
// mesmo trace; e os três passos de um workflow, cada um um job
// enfileirado pelo anterior, ficam todos no trace de quem iniciou o
// run.
func TestTraceAtravessaOBanco(t *testing.T) {
	gravador := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(gravador))
	antes := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTracerProvider(antes) })

	db := testutil.Postgres(t)
	h := api.NovaAPI(db, map[string]string{"t": "loja"},
		slog.New(slog.DiscardHandler)).Handler()
	const doCliente = "0af7651916cd43dd8448eb211c80319c"
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/jobs", bytes.NewReader([]byte(
			`{"queue":"default","kind":"eco","args":{}}`)))
	r.Header.Set("Authorization", "Bearer t")
	r.Header.Set("traceparent", "00-"+doCliente+"-b7ad6b7169203331-01")
	h.ServeHTTP(httptest.NewRecorder(), r)

	c := enxame.New(db, "loja")
	ctx, raiz := tp.Tracer("teste").Start(t.Context(), "checkout")
	runID, err := c.StartWorkflow(ctx, "tres-passos", "pedido-1", nil)
	raiz.End()
	if err != nil {
		t.Fatal(err)
	}

	w := c.NewWorker(enxame.WorkerConfig{
		Queues: map[string]int{"default": 2}})
	w.Handle("eco",
		func(context.Context, enxame.Job) error { return nil })
	w.Workflow("tres-passos", func(wc *workflow.Context,
		_ json.RawMessage) (any, error) {
		for _, p := range []string{"reservar", "cobrar", "enviar"} {
			if _, err := workflow.Step(wc, p,
				func(context.Context) (int, error) { return 1, nil },
			); err != nil {
				return nil, err
			}
		}
		return "ok", nil
	})
	rodando, parar := context.WithCancel(t.Context())
	fim := make(chan error, 1)
	go func() { fim <- w.Run(rodando) }()
	for inicio := time.Now(); ; {
		res, err := c.WorkflowResult(t.Context(), runID)
		if err == nil && res.State == "completed" {
			break
		}
		if time.Since(inicio) > 20*time.Second {
			t.Fatalf("o run não terminou: %+v %v", res, err)
		}
		<-time.After(50 * time.Millisecond)
	}
	parar()
	<-fim

	porTrace := map[string][]string{}
	for _, s := range gravador.Ended() {
		tid := s.SpanContext().TraceID().String()
		porTrace[tid] = append(porTrace[tid], s.Name())
	}
	t.Logf("spans do cliente HTTP: %v", porTrace[doCliente])
	t.Logf("spans do run: %v",
		porTrace[raiz.SpanContext().TraceID().String()])
	if !slices.Contains(porTrace[doCliente], "job eco") {
		t.Error("o job eco não está no trace da requisição")
	}
	passos := 0
	for _, n := range porTrace[raiz.SpanContext().TraceID().String()] {
		if n == "job "+workflow.AdvanceKind {
			passos++
		}
	}
	if passos < 3 {
		t.Errorf("%d tentativas do run no trace dele; esperava ao "+
			"menos 3", passos)
	}
}

// livro:fim teste-trace
