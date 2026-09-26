//go:build defeito

package keepalive_test

import (
	"net"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/examples/cap18/keepalive"
)

// O worker ocioso, com ping a cada 10 s, contra o servidor padrão.
func TestPingDemaisDerrubaOStream(t *testing.T) {
	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := keepalive.Servidor(lis, keepalive.Padrao)
	defer srv.Stop()
	inicio := time.Now()
	err = keepalive.Esperar(t.Context(), lis.Addr().String())
	t.Fatalf("stream encerrado depois de %v: %v",
		time.Since(inicio).Round(time.Second), err)
}
