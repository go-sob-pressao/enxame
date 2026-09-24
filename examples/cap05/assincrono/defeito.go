//go:build defeito

package main

import (
	"context"
	"net/http"
)

// livro:inicio assincrono-defeito

// aceitar responde 202 e processa o pedido em segundo plano, levando o
// contexto da requisição — que o net/http cancela assim que o handler
// retorna. No teste local, o processamento termina antes; em produção,
// não.
func aceitar(feito chan<- error) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(
				r.Context(),
				chaveRequisicao{},
				r.Header.Get("X-Request-Id"),
			)
			go processar(ctx, r.PathValue("id"), feito)
			w.WriteHeader(http.StatusAccepted)
		},
	)
}

// livro:fim assincrono-defeito
