//go:build !defeito

package main

import "net/http"

// handlerFrete: a versão corrigida entra no Capítulo 7, depois que o
// leitor tiver tentado encontrar o defeito sozinho. Até lá, rode com
// -tags defeito.
func handlerFrete(func(string) (int, error)) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"resolvido no Capítulo 7",
				http.StatusNotImplemented,
			)
		},
	)
}
