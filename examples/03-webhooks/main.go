// Command webhooks: publicar eventos na transação da aplicação e
// entregá-los, assinados, a um endpoint que cai e volta (Capítulo 20).
//
//	make up
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go run ./examples/03-webhooks
//
// O receptor responde 500 nos primeiros 6 segundos e 200 depois.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
)

// receptor é o endpoint do cliente: verifica a assinatura, anota o
// webhook-id, e fica fora do ar até recuperado.
type receptor struct {
	segredo    string
	inicio     time.Time
	recuperado time.Duration
	mu         sync.Mutex
	vistos     map[string]int
}

// livro:inicio receptor

func (r *receptor) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	corpo, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		http.Error(w, "corpo", http.StatusBadRequest)
		return
	}
	id, err := webhook.Verificar(r.segredo, req.Header, corpo,
		time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	desde := time.Since(r.inicio).Round(100 * time.Millisecond)
	if desde < r.recuperado {
		fmt.Printf("%5v  500  %s…\n", desde, id[:13])
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.mu.Lock()
	r.vistos[id]++
	repetida := r.vistos[id] > 1
	r.mu.Unlock()
	if repetida { // o webhook-id é a chave de deduplicação
		fmt.Printf("%5v  200  %s… (repetida, ignorada)\n", desde,
			id[:13])
	} else {
		fmt.Printf("%5v  200  %s…\n", desde, id[:13])
	}
	w.WriteHeader(http.StatusOK)
}

// livro:fim receptor

func main() {
	if err := rodar(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rodar(ctx context.Context) error {
	db, err := exemplo.Banco(ctx, "enxame_exemplo03")
	if err != nil {
		return err
	}
	defer db.Close()
	chave := make([]byte, 32)
	_, _ = rand.Read(chave)
	segredo := "whsec_" + base64.StdEncoding.EncodeToString(chave)
	if err := os.Setenv("SEGREDO_EXEMPLO_03", segredo); err != nil {
		return err
	}
	r := &receptor{segredo: segredo, recuperado: 6 * time.Second,
		vistos: map[string]int{}}
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r, ReadHeaderTimeout: time.Second}
	go func() { _ = srv.Serve(lis) }()
	defer srv.Close()

	c := enxame.New(db, "exemplo-03")
	if _, err := c.CreateEndpoint(ctx, enxame.Endpoint{
		URL:        "http://" + lis.Addr().String() + "/hooks",
		EventTypes: []string{"pedido.pago"},
		SecretRef:  "env:SEGREDO_EXEMPLO_03"}); err != nil {
		return err
	}
	r.inicio = time.Now()
	for i := range 3 {
		err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
			_, err := c.PublishTx(ctx, tx, webhook.Message{
				EventType: "pedido.pago",
				Payload:   map[string]int{"pedido": i + 1}})
			return err
		})
		if err != nil {
			return err
		}
	}
	fmt.Println("tempo  status  webhook-id")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go exemplo.Esperar(ctx, db, cancel)
	w := c.NewWorker(enxame.WorkerConfig{
		RetryBase: 500 * time.Millisecond, RetryMax: 4 * time.Second,
		BreakerLimiar: 3, BreakerPausa: 2 * time.Second})
	if err := w.Run(ctx); err != nil {
		return err
	}
	var chegaram, dosJobs int
	err = db.QueryRow(context.WithoutCancel(ctx), `SELECT
		(SELECT count(*) FROM webhook_attempt),
		(SELECT sum(attempt) FROM job WHERE kind = 'webhook.deliver')`,
	).Scan(&chegaram, &dosJobs)
	if err != nil {
		return err
	}
	fmt.Printf("%d tentativas dos jobs; %d chegaram ao endpoint; "+
		"%d mensagens entregues\n", dosJobs, chegaram, len(r.vistos))
	return nil
}
