package keepalive_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/examples/cap18/keepalive"
)

// Com a política alinhada, o mesmo worker passa 45 segundos ocioso — o
// tempo de quatro pings — sem perder o stream.
func TestPoliticaAlinhada(t *testing.T) {
	if testing.Short() {
		t.Skip("45 segundos")
	}
	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := keepalive.Servidor(lis, keepalive.Alinhada)
	defer srv.Stop()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	if err := keepalive.Esperar(ctx, lis.Addr().String()); err != nil {
		t.Fatalf("stream caiu: %v", err)
	}
}
