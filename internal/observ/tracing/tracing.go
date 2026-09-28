package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config diz para onde os spans vão e quantos são guardados.
type Config struct {
	Servico    string  // service.name: "enxamed", "worker"…
	No         string  // o nome do nó, como atributo do recurso
	Endpoint   string  // OTLP/HTTP, ex. "localhost:4318"; vazio: nada
	Amostragem float64 // fração dos traces novos que é guardada, 0 a 1
}

// livro:inicio iniciar

// Iniciar liga o OpenTelemetry no processo: um provedor de traces que
// manda os spans, em lotes, para um coletor OTLP; a amostragem decidida
// na raiz do trace e respeitada por todos os spans filhos, em qualquer
// processo; e a propagação W3C (traceparent) como padrão. Sem endpoint,
// não liga nada, e os spans são no-ops que não custam quase nada.
// Devolve quem desliga, esvaziando o que ficou no lote.
func Iniciar(ctx context.Context,
	c Config) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	if c.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(c.Endpoint),
		otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}
	rec, err := resource.Merge(resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", c.Servico),
			attribute.String("enxame.no", c.No)))
	if err != nil {
		return nil, errors.Join(err, exp.Shutdown(ctx))
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(rec),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(c.Amostragem))))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// livro:fim iniciar
