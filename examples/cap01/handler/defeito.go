//go:build defeito

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// livro:inicio handler-defeito

// handlerFrete responde em até 500 ms: se o frete não chegar a tempo,
// devolve 504 e segue a vida. O código passou em três revisões.
func handlerFrete(
	consultar func(context.Context, string) (int, error),
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			resultado := make(chan int)
			go func() {
				valor, _ := consultar(
					context.Background(),
					r.PathValue("cep"),
				)
				resultado <- valor
			}()

			select {
			case valor := <-resultado:
				fmt.Fprintf(w, "frete: %d\n", valor)
			case <-time.After(500 * time.Millisecond):
				http.Error(
					w,
					"frete indisponível",
					http.StatusGatewayTimeout,
				)
			}
		},
	)
}

// livro:fim handler-defeito
