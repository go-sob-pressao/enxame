// Command foradeordem é o incidente do Capítulo 22: o pedido.cancelado
// que chega ao cliente antes do pedido.criado.
//
//	make up
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go run ./examples/cap22/foradeordem -nos 1
//	go run ./examples/cap22/foradeordem -nos 2
//	go run ./examples/cap22/foradeordem -nos 1 -falhas 10
//
// Cada pedido publica dois eventos, em duas transações, nesta ordem:
// pedido.criado e pedido.cancelado. Cada "nó" é um pool de entrega com
// um worker só e o próprio pool de conexões com o banco — para o
// Postgres, dois nós no mesmo processo são indistinguíveis de dois
// processos em duas máquinas.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/internal/delivery"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
)

// livro:inicio receptor-ordem

// receptor é o sistema do cliente: anota a ordem em que os eventos de
// cada pedido chegam. Com falhas > 0, recusa com 500 a primeira
// tentativa do pedido.criado de um em cada falhas pedidos.
type receptor struct {
	falhas    int
	mu        sync.Mutex
	chegou    map[int][]string // pedido → tipos, na ordem de chegada
	recusados map[int]bool
}

func (r *receptor) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var ev struct {
		Type string `json:"type"`
		Data struct {
			Pedido int `json:"pedido"`
		} `json:"data"`
	}
	corpo, _ := io.ReadAll(req.Body)
	if err := json.Unmarshal(corpo, &ev); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p := ev.Data.Pedido
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.falhas > 0 && p%r.falhas == 0 && ev.Type == "pedido.criado" &&
		!r.recusados[p] {
		r.recusados[p] = true
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.chegou[p] = append(r.chegou[p], ev.Type)
	w.WriteHeader(http.StatusOK)
}

// foraDeOrdem conta os pedidos cujo cancelamento chegou primeiro.
func (r *receptor) foraDeOrdem() (n int, exemplos []int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for p, tipos := range r.chegou {
		if len(tipos) > 0 && tipos[0] == "pedido.cancelado" {
			n++
			if len(exemplos) < 5 {
				exemplos = append(exemplos, p)
			}
		}
	}
	return n, exemplos
}

// livro:fim receptor-ordem

func main() {
	nos := flag.Int("nos", 1, "nós entregando")
	pedidos := flag.Int("pedidos", 200, "pedidos, dois eventos cada")
	falhas := flag.Int("falhas", 0,
		"recusa a 1ª entrega do pedido.criado de 1 em cada N pedidos")
	flag.Parse()
	if err := rodar(context.Background(), *nos, *pedidos,
		*falhas); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rodar(ctx context.Context, nos, pedidos, falhas int) error {
	db, err := exemplo.Banco(ctx, "enxame_exemplo_cap22")
	if err != nil {
		return err
	}
	defer db.Close()
	chave := make([]byte, 32)
	_, _ = rand.Read(chave)
	if err := os.Setenv("SEGREDO_CAP22", "whsec_"+
		base64.StdEncoding.EncodeToString(chave)); err != nil {
		return err
	}
	r := &receptor{falhas: falhas, chegou: map[int][]string{},
		recusados: map[int]bool{}}
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r, ReadHeaderTimeout: time.Second}
	go func() { _ = srv.Serve(lis) }()
	defer srv.Close()

	c := enxame.New(db, "loja")
	if _, err := c.CreateEndpoint(ctx, enxame.Endpoint{
		URL:        "http://" + lis.Addr().String() + "/hooks",
		EventTypes: []string{"pedido.criado", "pedido.cancelado"},
		SecretRef:  "env:SEGREDO_CAP22"}); err != nil {
		return err
	}
	if err := publicar(ctx, db, c, pedidos); err != nil {
		return err
	}
	inicio := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	for i := range nos {
		g.Go(func() error {
			return no(ctx, fmt.Sprintf("no-%c", 'a'+i), db.Config())
		})
	}
	g.Go(func() error {
		exemplo.Esperar(ctx, db, cancel)
		return nil
	})
	if err := g.Wait(); err != nil && ctx.Err() == nil {
		return err
	}
	n, ex := r.foraDeOrdem()
	fmt.Printf("%d nó(s), %d pedidos: %d com o cancelamento antes da "+
		"criação %v — %v\n", nos, pedidos, n, ex,
		time.Since(inicio).Round(10*time.Millisecond))
	return nil
}

// publicar grava, para cada pedido, o criado e depois o cancelado —
// cada um no COMMIT da sua própria transação, como numa aplicação.
func publicar(ctx context.Context, db *pgxpool.Pool, c *enxame.Client,
	pedidos int) error {
	for p := 1; p <= pedidos; p++ {
		for _, tipo := range []string{"pedido.criado",
			"pedido.cancelado"} {
			err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
				_, err := c.PublishTx(ctx, tx, webhook.Message{
					EventType: tipo,
					Payload:   map[string]int{"pedido": p}})
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// livro:inicio no

// no é um nó de entrega: o próprio pool de conexões, um entregador e
// um worker só na fila de webhooks. Um nó sozinho entrega na ordem em
// que os jobs foram criados; dois nós, cada um na sua ordem.
func no(ctx context.Context, nome string, cfg *pgxpool.Config) error {
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	s := postgres.New(db)
	d := delivery.Novo(s)
	p := &worker.Pool{
		Queue: postgres.NewFila(ctx, s), QueueName: webhook.FanoutQueue,
		Handlers: map[string]runner.Handler{
			webhook.FanoutKind:  d.Fanout,
			webhook.DeliverKind: d.Entregar,
		},
		Concurrency: 1, PollTimeout: time.Second,
		AttemptTimeout: 10 * time.Second,
		RetryDelay:     200 * time.Millisecond,
		ReportEvery:    time.Hour, Worker: nome, Now: time.Now,
		Log: slog.New(slog.DiscardHandler),
	}
	return p.Run(ctx)
}

// livro:fim no
