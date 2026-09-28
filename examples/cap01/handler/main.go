// Command handler: o serviço que caiu no 11º dia (O bug que parecia
// impossível #1). O defeito é apresentado no Capítulo 1 e resolvido no
// 7.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// consultarFrete simula um serviço externo lento.
func consultarFrete(ctx context.Context, cep string) (int, error) {
	select {
	case <-time.After(2 * time.Second):
		return len(cep) * 100, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	http.Handle("GET /frete/{cep}", handlerFrete(consultarFrete))
	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
	}
	ctx := context.Background()
	log.InfoContext(ctx, "ouvindo", slog.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil {
		log.ErrorContext(ctx, "servidor", slog.Any("erro", err))
	}
}
