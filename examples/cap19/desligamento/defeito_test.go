//go:build defeito

package desligamento_test

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/examples/cap19/desligamento"
)

// livro:inicio desligamento-teste

// Um cliente está no long-poll. O orquestrador manda SIGTERM; o
// servidor dá 2 segundos ao Shutdown — no incidente, eram 25 de 30.
func TestShutdownEsperaOLongPoll(t *testing.T) {
	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := desligamento.Servidor(lis, make(chan struct{}))
	go func() {
		req, _ := http.NewRequestWithContext(t.Context(),
			http.MethodGet, "http://"+lis.Addr().String(), nil)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
	<-time.After(200 * time.Millisecond) // o cliente está esperando
	inicio := time.Now()
	err = desligamento.Desligar(srv, 2*time.Second)
	t.Fatalf("Shutdown voltou em %v: %v",
		time.Since(inicio).Round(10*time.Millisecond), err)
}

// livro:fim desligamento-teste
