package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

const nome = "github.com/go-sob-pressao/enxame"

// livro:inicio persistencia

// TraceParent devolve o contexto de trace de ctx no formato W3C, para
// gravar no job; vazio se não houver span.
func TraceParent(ctx context.Context) string {
	c := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, c)
	return c["traceparent"]
}

// IniciarJob abre o span de uma tentativa de job como filho do contexto
// que foi gravado com ele: o span pertence ao trace de quem enfileirou,
// mesmo que isso tenha acontecido em outro processo, ou dias antes. O
// span termina com o erro da tentativa, se houver.
func IniciarJob(ctx context.Context,
	j job.Job) (context.Context, func(error)) {
	pai := otel.GetTextMapPropagator().Extract(ctx,
		propagation.MapCarrier{"traceparent": j.TraceParent})
	//nolint:spancheck // quem chama encerra, pela função devolvida
	ctx, span := otel.Tracer(nome).Start(pai, "job "+j.Kind,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("enxame.job.id", j.ID.String()),
			attribute.String("enxame.job.fila", j.Queue),
			attribute.Int("enxame.job.tentativa", j.Attempt),
			attribute.Int("enxame.job.particao", j.Particao())))
	return ctx, func(err error) { //nolint:spancheck // End abaixo
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// livro:fim persistencia

// livro:inicio requisicao

// IniciarRequisicao abre o span de servidor de uma requisição HTTP,
// filho do traceparent que o cliente mandou, se mandou. O nome é a
// rota, não o caminho: /v1/jobs/{id}, e não um nome por job.
func IniciarRequisicao(ctx context.Context, h http.Header,
	rota string) (context.Context, func(status int)) {
	ctx = otel.GetTextMapPropagator().Extract(ctx,
		propagation.HeaderCarrier(h))
	//nolint:spancheck // quem chama encerra, pela função devolvida
	ctx, span := otel.Tracer(nome).Start(ctx, rota,
		trace.WithSpanKind(trace.SpanKindServer))
	return ctx, func(status int) { //nolint:spancheck // End abaixo
		span.SetAttributes(
			attribute.Int("http.response.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(status))
		}
		span.End()
	}
}

// livro:fim requisicao
