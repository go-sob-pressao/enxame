// Package custo — quanto custa o tracing de um job (Capítulo 30).
package custo

import (
	"context"
	"testing"
	"time"
	"uuid"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/observ/tracing"
)

// livro:inicio custo-tracing

// O que o tracing acrescenta a cada job: gravar o contexto ao
// enfileirar e abrir e fechar o span da tentativa. Três provedores: o
// no-op (sem -otlp), o SDK guardando todos os traces, e o SDK guardando
// um em cem. Os spans guardados vão para um exportador que os descarta,
// em lote, como o OTLP faria com a rede.
func BenchmarkTracingPorJob(b *testing.B) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	descarte := sdktrace.WithBatcher(tracetest.NewNoopExporter())
	for _, c := range []struct {
		nome  string
		fazer func() func()
	}{
		{"sem-provedor", func() func() {
			otel.SetTracerProvider(noop.NewTracerProvider())
			return func() {}
		}},
		{"amostragem-100%", provedor(1, descarte)},
		{"amostragem-1%", provedor(0.01, descarte)},
	} {
		b.Run(c.nome, func(b *testing.B) {
			desfazer := c.fazer()
			defer desfazer()
			j := job.Job{ID: id.JobID(uuid.NewV7()), Queue: "q",
				Kind: "cobrar", Attempt: 1}
			b.ReportAllocs()
			for b.Loop() {
				ctx, fim := tracing.IniciarRequisicao(
					context.Background(), nil, "POST")
				j.TraceParent = tracing.TraceParent(ctx)
				fim("POST /v1/jobs", 201)
				_, fimJob := tracing.IniciarJob(context.Background(), j)
				fimJob(nil)
			}
		})
	}
}

// livro:fim custo-tracing

func provedor(fracao float64,
	o sdktrace.TracerProviderOption) func() func() {
	return func() func() {
		tp := sdktrace.NewTracerProvider(o, sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(fracao))))
		otel.SetTracerProvider(tp)
		return func() {
			ctx, cancelar := context.WithTimeout(context.Background(),
				5*time.Second)
			defer cancelar()
			_ = tp.Shutdown(ctx)
		}
	}
}
