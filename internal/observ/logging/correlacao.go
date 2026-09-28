package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	pkgjob "github.com/go-sob-pressao/enxame/pkg/job"
)

// livro:inicio correlacao

// Correlacao acrescenta a cada linha de log o que liga essa linha ao
// resto da telemetria: o trace_id e o span_id do span corrente, e o id,
// o kind e a tentativa do job que está rodando. Com eles, de um trace
// se chega às linhas que ele escreveu, e de uma linha, ao trace.
type Correlacao struct{ slog.Handler }

// Handle anota o registro e o passa adiante.
func (c Correlacao) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()))
	}
	if j, ok := pkgjob.FromContext(ctx); ok {
		r.AddAttrs(slog.String("job_id", j.ID),
			slog.String("kind", j.Kind),
			slog.Int("tentativa", j.Attempt))
	}
	return c.Handler.Handle(ctx, r)
}

// livro:fim correlacao

// WithAttrs mantém a correlação nos loggers derivados.
func (c Correlacao) WithAttrs(as []slog.Attr) slog.Handler {
	return Correlacao{Handler: c.Handler.WithAttrs(as)}
}

// WithGroup mantém a correlação nos loggers derivados.
func (c Correlacao) WithGroup(nome string) slog.Handler {
	return Correlacao{Handler: c.Handler.WithGroup(nome)}
}
