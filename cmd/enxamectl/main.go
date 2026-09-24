// Command enxamectl é a CLI de operação do Enxame.
//
//	enxamectl job insert --queue padrao --kind email.enviar \
//	    --args @args.json
//	enxamectl job describe     --id 0199…
//	enxamectl job retry        --id 0199…
//	enxamectl workflow start --type ProcessarPedido --id pedido-42 \
//	    --input @pedido.json
//	enxamectl workflow history --id pedido-42 --follow
//	enxamectl webhook endpoint add \
//	    --url https://cliente.exemplo/hooks \
//	    --events pedido.pago
//	enxamectl webhook attempts --message 0199…
//	enxamectl schedule list
//	enxamectl partition list
//	enxamectl cluster status
package main

import (
	"fmt"
	"io"
	"os"
)

// versao é preenchida no build: -ldflags "-X main.versao=v1.2.3".
var versao = "dev"

func main() {
	os.Exit(executar(os.Args[1:], os.Stdout, os.Stderr))
}

func executar(args []string, saida, erros io.Writer) int {
	if len(args) == 1 &&
		(args[0] == "version" || args[0] == "-version") {
		fmt.Fprintln(saida, "enxamectl", versao)
		return 0
	}
	// Cap. 19: a CLI fala com a API HTTP pública.
	fmt.Fprintf(
		erros,
		"enxamectl %s: esqueleto do cap-00 — "+
			"os comandos entram no Capítulo 19\n",
		versao,
	)
	return 1
}
