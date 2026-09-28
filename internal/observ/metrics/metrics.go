package metrics

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registro guarda todas as métricas do processo: as do Enxame e as do
// runtime do Go e do processo.
var Registro = prometheus.NewRegistry()

func init() {
	Registro.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(
			collectors.ProcessCollectorOpts{}))
}

// Servir expõe /metrics em lis, até ctx terminar.
func Servir(ctx context.Context, lis net.Listener) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(Registro,
		promhttp.HandlerOpts{}))
	srv := &http.Server{Handler: mux,
		ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	if err := srv.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
