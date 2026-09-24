//go:build !defeito

package main

import (
	"context"
	"net/http"
	"time"
)

// livro:inicio assincrono-correto

// aceitar desacopla o ciclo de vida: WithoutCancel mantém os valores do
// contexto (o request id segue nos logs) e descarta o cancelamento; o
// trabalho ganha um prazo próprio, porque todo trabalho precisa de um.
func aceitar(feito chan<- error) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(
				r.Context(),
				chaveRequisicao{},
				r.Header.Get("X-Request-Id"),
			)
			go func() {
				ctx, cancel := context.WithTimeout(
					context.WithoutCancel(ctx),
					5*time.Minute,
				)
				defer cancel()
				processar(ctx, r.PathValue("id"), feito)
			}()
			w.WriteHeader(http.StatusAccepted)
		},
	)
}

// livro:fim assincrono-correto
