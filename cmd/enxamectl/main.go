// Command enxamectl é a CLI de operação do Enxame, sobre a API HTTP.
//
//	enxamectl job insert --queue padrao --kind email.enviar \
//	    --args @args.json
//	enxamectl job describe --id 0199… [--wait 30s]
//	enxamectl job cancel --id 0199…
//	enxamectl workflow start --type pedido --id pedido-42 \
//	    --input @pedido.json
//	enxamectl workflow describe --run 0199…
//	enxamectl webhook endpoint add \
//	    --url https://cliente.exemplo/hooks \
//	    --events pedido.pago --secret-ref env:SEGREDO
//	enxamectl webhook endpoint list
//	enxamectl webhook endpoint remove --id 0199…
//	enxamectl webhook endpoint state --id 0199…
//	enxamectl cluster members
//
// A API vem de ENXAME_API (padrão http://localhost:8080); o token, de
// ENXAME_TOKEN.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// versao é preenchida no build: -ldflags "-X main.versao=v1.2.3".
var versao = "dev"

func main() {
	os.Exit(executar(os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

// comando é um subcomando da CLI: recebe os argumentos que sobram.
type comando func(c *cliente, args []string) error

// livro:inicio ctl-comandos

var comandos = map[string]comando{
	"job insert":              jobInsert,
	"job describe":            jobDescribe,
	"job cancel":              jobCancel,
	"workflow start":          workflowStart,
	"workflow describe":       workflowDescribe,
	"webhook endpoint add":    endpointAdd,
	"webhook endpoint list":   endpointList,
	"webhook endpoint remove": endpointRemove,
	"webhook endpoint state":  endpointState,
	"cluster members":         clusterMembers,
}

// executar acha o comando mais longo que casa com o começo dos
// argumentos, e o executa. getenv vem de fora, para os testes.
func executar(
	args []string,
	saida, erros io.Writer,
	getenv func(string) string,
) int {
	if len(args) == 1 &&
		(args[0] == "version" || args[0] == "-version") {
		fmt.Fprintln(saida, "enxamectl", versao)
		return 0
	}
	for n := min(len(args), 3); n > 0; n-- {
		cmd, ok := comandos[strings.Join(args[:n], " ")]
		if !ok {
			continue
		}
		api := getenv("ENXAME_API")
		if api == "" {
			api = "http://localhost:8080"
		}
		c := &cliente{api: api, token: getenv("ENXAME_TOKEN"),
			saida: saida}
		if err := cmd(c, args[n:]); err != nil {
			fmt.Fprintln(erros, "enxamectl:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintln(erros, "enxamectl: comando desconhecido; veja o "+
		"cabeçalho de cmd/enxamectl/main.go")
	return 2
}

// livro:fim ctl-comandos
