// Command openapi gera api/openapi.json a partir das rotas da API.
//
//	go run ./tools/openapi > api/openapi.json   (make openapi)
package main

import (
	"fmt"
	"log/slog"
	"os"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

func main() {
	a := api.NovaAPI(nil, nil, slog.New(slog.DiscardHandler))
	b, err := a.OpenAPI("v1")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}
