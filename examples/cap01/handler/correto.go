//go:build !defeito

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// livro:inicio handler-correto

// handlerFrete responde em até 500 ms. A goroutine tem dono e critério
// de término: o canal tem buffer de 1 — o envio completa mesmo depois
// do 504 — e a consulta recebe o contexto da requisição com prazo,
// então para de trabalhar quando ninguém mais espera por ela.
func handlerFrete(
	consultar func(context.Context, string) (int, error),
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(
				r.Context(),
				500*time.Millisecond,
			)
			defer cancel()

			resultado := make(chan int, 1)
			go func() {
				valor, _ := consultar(ctx, r.PathValue("cep"))
				resultado <- valor
			}()

			select {
			case valor := <-resultado:
				fmt.Fprintf(w, "frete: %d\n", valor)
			case <-ctx.Done():
				http.Error(
					w,
					"frete indisponível",
					http.StatusGatewayTimeout,
				)
			}
		},
	)
}

// livro:fim handler-correto
