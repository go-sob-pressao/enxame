package custo_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-sob-pressao/enxame/examples/cap18/custo"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
)

const jobID = "0192a3b4-0000-7000-8000-000000000001"

// livro:inicio custo-grpc

func BenchmarkGRPC(b *testing.B) { grpcBench(b, true) }

// Sem os interceptors de token, prazo e erros: o custo do transporte.
func BenchmarkGRPCSemInterceptors(b *testing.B) { grpcBench(b, false) }

func grpcBench(b *testing.B, interceptors bool) {
	var lc net.ListenConfig
	lis, err := lc.Listen(b.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	var sopts []grpc.ServerOption
	copts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials())}
	if interceptors {
		sopts = tgrpc.Servidor("t", slog.New(slog.DiscardHandler))
		copts = append(copts, tgrpc.Cliente("t", time.Second)...)
	}
	srv := grpc.NewServer(sopts...)
	enxamev1.RegisterWorkerServiceServer(srv, &tgrpc.Server{
		Motor: custo.Motor{}, Poll: time.Second, Now: time.Now})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()
	conn, err := grpc.NewClient(lis.Addr().String(), copts...)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	c := enxamev1.NewWorkerServiceClient(conn)
	req := &enxamev1.HeartbeatRequest{JobId: jobID, Attempt: 1}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := c.Heartbeat(b.Context(), req); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkHTTPJSON(b *testing.B) {
	srv := httptest.NewServer(custo.HandlerHTTP(custo.Motor{}))
	defer srv.Close()
	cliente := srv.Client()
	cliente.Transport.(*http.Transport).MaxIdleConnsPerHost = 64
	url := srv.URL + "/v1/heartbeat"
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			corpo, _ := json.Marshal(custo.Batimento{JobID: jobID,
				Attempt: 1})
			req, _ := http.NewRequestWithContext(b.Context(),
				http.MethodPost, url, bytes.NewReader(corpo))
			req.Header.Set("Content-Type", "application/json")
			resp, err := cliente.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})
}

// livro:fim custo-grpc
