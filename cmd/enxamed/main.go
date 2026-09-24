// Command enxamed é o servidor do Enxame.
//
// Um único binário hospeda os papéis — api, worker, scheduler, delivery
// e cluster. Quais ficam ativos é decidido pela configuração (ADR-008).
//
//	enxamed -demo 5     executa o M0: cinco jobs, um processo, um por vez
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

// livro:inicio main-testavel

func main() {
	os.Exit(executar(os.Args[1:], os.Stdout, os.Stderr))
}

// executar existe separado de main para ser testável: recebe os
// argumentos e as saídas, devolve o código de saída.
func executar(args []string, saida, erros io.Writer) int {
	fs := flag.NewFlagSet("enxamed", flag.ContinueOnError)
	fs.SetOutput(erros)
	config := fs.String("config", "", "arquivo de configuração do nó")
	demo := fs.Int("demo", 0, "executa N jobs de exemplo no M0 e sai")
	mostrarVersao := fs.Bool("version", false, "imprime a versão e sai")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *mostrarVersao {
		fmt.Fprintln(saida, "enxamed", versao)
		return 0
	}

	// livro:fim main-testavel
	if *demo > 0 {
		return demonstrar(*demo, saida, erros)
	}

	// Cap. 18: modo servidor, com workers remotos.
	// Cap. 19: encerramento gracioso no SIGTERM.
	// Cap. 32: drenagem de partições no preStop.
	fmt.Fprintf(
		erros,
		"enxamed %s: o modo servidor entra no Capítulo 18; "+
			"até lá, use -demo N (config: %q)\n",
		versao,
		*config,
	)
	return 1
}
