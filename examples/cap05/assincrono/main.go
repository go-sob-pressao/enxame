// Command assincrono: o job que falhava com "context canceled" só em
// produção.
//
//	go test -tags defeito ./examples/cap05/assincrono    reproduz
//	go test ./examples/cap05/assincrono                  a correção
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type chaveRequisicao struct{}

// RequestID devolve o identificador da requisição guardado no contexto.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(chaveRequisicao{}).(string)
	return id
}

// processar simula o trabalho demorado: respeita o contexto, como deve.
func processar(ctx context.Context, pedido string, feito chan<- error) {
	select {
	case <-time.After(200 * time.Millisecond):
		feito <- nil
	case <-ctx.Done():
		feito <- ctx.Err()
	}
	_ = pedido
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	feito := make(chan error, 100)
	http.Handle("POST /pedidos/{id}", aceitar(feito))
	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("servidor", slog.Any("erro", err))
	}
}
