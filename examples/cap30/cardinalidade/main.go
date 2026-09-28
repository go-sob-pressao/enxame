// Command cardinalidade serve, em /metrics, o contador de um painel
// "por job": um rótulo job_id, e mil jobs novos por segundo — o enigma
// do Capítulo 30. Com -certo, o mesmo contador rotulado só pela fila.
//
//	go run ./examples/cap30/cardinalidade -addr :9092
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// livro:inicio cardinalidade

// O painel pedia "tentativas de cada job". Cada job_id é uma série nova
// no Prometheus, para sempre — ou até o fim da retenção.
var porJob = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "demo_tentativas_por_job_total",
	Help: "Tentativas, por job (não faça isto).",
}, []string{"fila", "job_id"})

// livro:fim cardinalidade

var porFila = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "demo_tentativas_total",
	Help: "Tentativas, por fila.",
}, []string{"fila"})

func main() {
	addr := flag.String("addr", ":9092", "endereço do /metrics")
	porSegundo := flag.Int("jobs", 1000, "jobs novos por segundo")
	certo := flag.Bool("certo", false, "rotular só pela fila")
	flag.Parse()
	reg := prometheus.NewRegistry()
	reg.MustRegister(porJob, porFila)
	ctx, parar := signal.NotifyContext(context.Background(),
		os.Interrupt)
	defer parar()
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		n := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			for range *porSegundo / 10 {
				n++
				if *certo {
					porFila.WithLabelValues("cobranca").Inc()
				} else {
					porJob.WithLabelValues("cobranca",
						fmt.Sprintf("job-%09d", n)).Inc()
				}
			}
		}
	}()
	srv := &http.Server{Addr: *addr, ReadHeaderTimeout: time.Second,
		Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{})}
	go func() { <-ctx.Done(); _ = srv.Close() }()
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
