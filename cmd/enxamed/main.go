// Command enxamed é o servidor do Enxame.
//
// Um único binário hospeda os papéis — api, worker, scheduler, delivery
// e cluster. Quais ficam ativos é decidido pela configuração (ADR-008).
//
//	enxamed -dsn postgres://…    modo servidor: API HTTP, gRPC dos
//	                             workers remotos e o motor
//	enxamed -demo 5              executa o M0: cinco jobs, um por vez
//	enxamed -version
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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
	demo := fs.Int("demo", 0, "executa N jobs de exemplo no M0 e sai")
	mostrarVersao := fs.Bool("version", false, "imprime a versão e sai")
	var c config
	fs.StringVar(&c.dsn, "dsn", os.Getenv("ENXAME_DB_DSN"),
		"PostgreSQL (ENXAME_DB_DSN)")
	fs.StringVar(&c.http, "http", ":8080", "endereço da API HTTP")
	fs.StringVar(&c.grpc, "grpc", ":7233",
		"endereço do gRPC dos workers")
	tokens := fs.String("tokens", os.Getenv("ENXAME_TOKENS"),
		"token:namespace,… da API (ENXAME_TOKENS)")
	fs.StringVar(&c.tokenWorker, "worker-token",
		os.Getenv("ENXAME_WORKER_TOKEN"),
		"token dos workers remotos (ENXAME_WORKER_TOKEN)")
	fs.DurationVar(&c.aviso, "aviso", 5*time.Second,
		"tempo de /readyz em 503 antes de parar de aceitar")
	fs.DurationVar(&c.prazo, "prazo", 20*time.Second,
		"teto para drenar as requisições no desligamento")
	fs.DurationVar(&c.resgate, "resgate", time.Minute,
		"prazo sem sinal de vida antes de resgatar uma tentativa")
	fs.Float64Var(&c.taxa, "taxa", 0,
		"requisições por segundo por namespace (0: sem limite)")
	fs.Float64Var(&c.rajada, "rajada", 0,
		"rajada por namespace acima da taxa")
	fs.IntVar(&c.emCurso, "max-em-curso", 32,
		"requisições em curso antes de responder 503 (0: sem limite)")
	fs.IntVar(&c.naFila, "max-na-fila", 100000,
		"jobs esperando por namespace antes de responder 429 "+
			"(0: sem limite)")
	fs.DurationVar(&c.leaseMotor, "lease", 3*time.Second,
		"posse de cada partição sem renovação")
	fs.StringVar(&c.no, "no", "",
		"nome deste nó no cluster (padrão: máquina-pid)")
	fs.DurationVar(&c.vooLimiar, "voo-limiar", 0,
		"grava o trace dos últimos 10 s quando uma requisição passa "+
			"disto (0: desligado)")
	fs.StringVar(&c.vooDir, "voo-dir", os.TempDir(),
		"diretório dos traces do flight recorder")
	kinds := fs.String("kinds", os.Getenv("ENXAME_KINDS"),
		"kinds aceitos pela API, separados por vírgula (vazio: todos)")
	fs.StringVar(&c.diag, "diag", "",
		"endereço dos perfis do pprof (desligado se vazio; use "+
			"127.0.0.1:6060)")
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
	if *kinds != "" {
		c.kinds = strings.Split(*kinds, ",")
	}
	var err error
	if c.relogio, err = relogioDoNo(); err != nil {
		fmt.Fprintf(erros, "enxamed: %v\n", err)
		return 2
	}
	if c.tokens, err = lerTokens(*tokens); err != nil ||
		c.dsn == "" || c.tokenWorker == "" {
		fmt.Fprintf(erros, "enxamed %s: modo servidor precisa de "+
			"-dsn, -tokens e -worker-token (ou use -demo N): %v\n",
			versao, err)
		return 2
	}
	// SIGTERM é o pedido de desligamento do orquestrador; SIGINT, o
	// Ctrl+C de quem roda à mão.
	ctx, parar := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, os.Interrupt)
	defer parar()
	log := slog.New(slog.NewJSONHandler(erros, nil))
	var lc net.ListenConfig
	lisHTTP, err := lc.Listen(ctx, "tcp", c.http)
	if err == nil {
		var lisGRPC net.Listener
		if lisGRPC, err = lc.Listen(ctx, "tcp", c.grpc); err == nil {
			err = servir(ctx, c, lisHTTP, lisGRPC, log)
		}
	}
	if err != nil {
		log.ErrorContext(ctx, "enxamed", slog.Any("erro", err))
		return 1
	}
	return 0
}

// lerTokens lê "token:namespace,token:namespace".
func lerTokens(s string) (map[string]string, error) {
	m := map[string]string{}
	for par := range strings.SplitSeq(s, ",") {
		tk, ns, ok := strings.Cut(strings.TrimSpace(par), ":")
		if !ok || tk == "" || ns == "" {
			return nil, fmt.Errorf("token inválido: %q", par)
		}
		m[tk] = ns
	}
	return m, nil
}

// livro:inicio relogio-do-no

// relogioDoNo é o relógio de parede que o enxamed lê para decidir. O
// laboratório de caos (Cap. 28) o desloca com ENXAME_CAOS_DESVIO —
// "3m", "-90s" —, para simular um nó com o relógio errado sem mexer
// no relógio da máquina. Os intervalos (tickers, prazos) continuam no
// relógio monotônico, como numa máquina com o NTP quebrado.
func relogioDoNo() (func() time.Time, error) {
	v := os.Getenv("ENXAME_CAOS_DESVIO")
	if v == "" {
		return time.Now, nil
	}
	desvio, err := time.ParseDuration(v)
	if err != nil {
		return nil, fmt.Errorf("ENXAME_CAOS_DESVIO: %w", err)
	}
	return func() time.Time { return time.Now().Add(desvio) }, nil
}

// livro:fim relogio-do-no
