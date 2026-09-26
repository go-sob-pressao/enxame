//go:build integration

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/poller"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// O enxamed inteiro: um job entra pela API HTTP, um worker remoto o
// executa por gRPC, a API mostra o resultado; depois, o SIGTERM, com o
// stream do worker ainda aberto.
func TestServidorDePontaAPonta(t *testing.T) {
	c := config{dsn: testutil.PostgresDSN(t),
		tokens: map[string]string{"tk": "loja"}, tokenWorker: "tw",
		aviso: 100 * time.Millisecond, prazo: 5 * time.Second,
		resgate: time.Minute}
	var lc net.ListenConfig
	lisHTTP, _ := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	lisGRPC, _ := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	ctx, sigterm := context.WithCancel(t.Context())
	fim := make(chan error, 1)
	go func() {
		fim <- servir(ctx, c, lisHTTP, lisGRPC,
			slog.New(slog.DiscardHandler))
	}()
	base := "http://" + lisHTTP.Addr().String()
	chamar := func(metodo, caminho, corpo string) map[string]any {
		req, _ := http.NewRequestWithContext(t.Context(), metodo,
			base+caminho, strings.NewReader(corpo))
		req.Header.Set("Authorization", "Bearer tk")
		for range 50 { // o servidor pode ainda estar subindo
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				return m
			}
			<-time.After(50 * time.Millisecond)
			req, _ = http.NewRequestWithContext(t.Context(), metodo,
				base+caminho, strings.NewReader(corpo))
			req.Header.Set("Authorization", "Bearer tk")
		}
		t.Fatal("API fora do ar")
		return nil
	}
	j := chamar("POST", "/v1/jobs", `{"queue":"q","kind":"eco"}`)
	jid, _ := j["id"].(string)

	opts := append(tgrpc.Cliente("tw", time.Second),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(lisGRPC.Addr().String(), opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	wctx, wcancel := context.WithCancel(t.Context())
	defer wcancel()
	r, err := poller.Conectar(wctx, conn, "w1", []string{"q"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	p := &worker.Pool{Queue: r, QueueName: "q", Concurrency: 1,
		Handlers: map[string]runner.Handler{
			"eco": func(context.Context, job.Job) error { return nil }},
		PollTimeout: time.Second, AttemptTimeout: time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Worker: "w1", Now: time.Now, Log: slog.New(slog.DiscardHandler)}
	go func() { _ = p.Run(wctx) }()

	j = chamar("GET", "/v1/jobs/"+jid+"?wait=5s", "")
	if j["state"] != "completed" {
		t.Fatalf("job: %v", j)
	}
	inicio := time.Now()
	sigterm()
	if err := <-fim; err != nil {
		t.Fatal(err)
	}
	t.Logf("desligou em %v, com o stream do worker aberto",
		time.Since(inicio).Round(10*time.Millisecond))
	if time.Since(inicio) > 2*time.Second {
		t.Fatal("o desligamento esperou o stream do worker")
	}
}
