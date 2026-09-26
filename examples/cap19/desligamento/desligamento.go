// Package desligamento é o enigma do Capítulo 19: o Shutdown que
// espera um long-poll.
package desligamento

import (
	"context"
	"net"
	"net/http"
	"time"
)

// livro:inicio desligamento-ingenuo

// Espera é um long-poll ingênuo: segura a requisição até o evento
// chegar ou o cliente desistir. Não sabe que o servidor quer desligar.
func Espera(evento <-chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-evento:
			w.WriteHeader(http.StatusOK)
		case <-time.After(30 * time.Second):
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
		}
	}
}

// Desligar faz o que o manual manda: Shutdown com um prazo que cabe no
// tempo que o orquestrador dá antes do SIGKILL.
func Desligar(srv *http.Server, prazo time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), prazo)
	defer cancel()
	return srv.Shutdown(ctx)
}

// livro:fim desligamento-ingenuo

// Servidor sobe o long-poll ingênuo em lis.
func Servidor(lis net.Listener, evento <-chan struct{}) *http.Server {
	srv := &http.Server{Handler: Espera(evento),
		ReadHeaderTimeout: time.Second}
	go func() { _ = srv.Serve(lis) }()
	return srv
}
