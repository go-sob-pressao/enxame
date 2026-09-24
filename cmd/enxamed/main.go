// Command enxamed é o servidor do Enxame.
//
// Um único binário hospeda os papéis — api, worker, scheduler, delivery
// e cluster. Quais ficam ativos é decidido pela configuração (ADR-008).
//
//	enxamed -config deploy/docker/enxamed.dev.yaml
//	enxamed -version
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// versao é preenchida no build: -ldflags "-X main.versao=v1.2.3".
var versao = "dev"

func main() {
	os.Exit(executar(os.Args[1:], os.Stdout, os.Stderr))
}

// executar existe separado de main para ser testável: recebe os
// argumentos e as saídas, devolve o código de saída.
func executar(args []string, saida, erros io.Writer) int {
	fs := flag.NewFlagSet("enxamed", flag.ContinueOnError)
	fs.SetOutput(erros)
	config := fs.String("config", "", "arquivo de configuração do nó")
	mostrarVersao := fs.Bool("version", false, "imprime a versão e sai")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *mostrarVersao {
		fmt.Fprintln(saida, "enxamed", versao)
		return 0
	}

	// Cap. 2: carregamento da configuração e wiring dos papéis.
	// Cap. 19: encerramento gracioso no SIGTERM.
	// Cap. 32: drenagem de partições no preStop.
	fmt.Fprintf(
		erros,
		"enxamed %s: esqueleto do cap-00 — os papéis entram a partir do Capítulo 2 (config: %q)\n",
		versao,
		*config,
	)
	return 1
}
