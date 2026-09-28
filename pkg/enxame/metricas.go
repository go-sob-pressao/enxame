package enxame

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-sob-pressao/enxame/internal/observ/metrics"
)

// MetricsHandler serve as métricas do Enxame neste processo, no formato
// do Prometheus: no modo biblioteca, as tentativas, o atraso e as filas
// são contados no processo da aplicação, e só ela pode expô-los
// (Cap. 30).
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(metrics.Registro, promhttp.HandlerOpts{})
}
