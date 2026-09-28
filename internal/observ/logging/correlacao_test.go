package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-sob-pressao/enxame/internal/observ/logging"
	pkgjob "github.com/go-sob-pressao/enxame/pkg/job"
)

func TestCorrelacao(t *testing.T) {
	var saida bytes.Buffer
	log := slog.New(logging.Correlacao{
		Handler: slog.NewJSONHandler(&saida, nil)})
	tid, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	sid, _ := trace.SpanIDFromHex("b7ad6b7169203331")
	ctx := trace.ContextWithSpanContext(context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid,
			SpanID: sid}))
	ctx = pkgjob.WithInfo(ctx, pkgjob.Info{ID: "j1", Kind: "eco",
		Attempt: 2})
	log.With(slog.String("no", "no-1")).InfoContext(ctx, "cobrando")
	var linha map[string]any
	if err := json.Unmarshal(saida.Bytes(), &linha); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]any{"trace_id": tid.String(),
		"span_id": sid.String(), "job_id": "j1", "kind": "eco",
		"tentativa": 2.0, "no": "no-1"} {
		if linha[k] != v {
			t.Errorf("%s = %v; esperava %v", k, linha[k], v)
		}
	}
}
