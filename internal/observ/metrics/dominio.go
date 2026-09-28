package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Os limites dos histogramas de tempo, de 5 ms a 1 h: o atraso de um
// job vai de nada a horas, e um balde por ordem de grandeza basta.
var baldes = []float64{.005, .025, .1, .5, 1, 2.5, 5, 15, 60, 300,
	900, 3600}

// livro:inicio dominio

var (
	atraso = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "enxame_job_atraso_segundos",
		Help:    "Da hora marcada do job ao começo da tentativa.",
		Buckets: baldes,
	}, []string{"fila"})
	duracao = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "enxame_job_duracao_segundos",
		Help:    "Do começo ao fim de cada tentativa.",
		Buckets: baldes,
	}, []string{"fila", "resultado"})
	tentativas = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "enxame_job_tentativas_total",
		Help: "Tentativas terminadas, por resultado: concluido, " +
			"retry, descartado.",
	}, []string{"fila", "resultado"})
)

// JobReservado anota o atraso de uma tentativa que começou: quanto o
// job esperou depois da hora em que devia rodar. É a métrica que diz se
// o Enxame está dando conta, antes de a fila crescer.
func JobReservado(fila string, marcado, inicio time.Time) {
	atraso.WithLabelValues(fila).Observe(
		max(0, inicio.Sub(marcado).Seconds()))
}

// JobTerminou anota o fim de uma tentativa. O resultado é um de três
// valores; o id do job, jamais: cada id seria uma série nova, para
// sempre (Anti-Pattern #19).
func JobTerminou(fila, resultado string, inicio, fim time.Time) {
	tentativas.WithLabelValues(fila, resultado).Inc()
	duracao.WithLabelValues(fila, resultado).Observe(
		max(0, fim.Sub(inicio).Seconds()))
}

// livro:fim dominio

var (
	requisicoes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "enxame_http_requisicoes_total",
		Help: "Requisições da API, por rota e status.",
	}, []string{"rota", "status"})
	latencia = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "enxame_http_duracao_segundos",
		Help:    "Duração das requisições da API, por rota.",
		Buckets: prometheus.DefBuckets,
	}, []string{"rota"})
)

// Requisicao anota uma requisição: a taxa, os erros e a duração (RED).
func Requisicao(rota string, status int, d time.Duration) {
	if rota == "" {
		rota = "sem rota"
	}
	requisicoes.WithLabelValues(rota, strconv.Itoa(status)).Inc()
	latencia.WithLabelValues(rota).Observe(d.Seconds())
}

func init() {
	Registro.MustRegister(atraso, duracao, tentativas, requisicoes,
		latencia)
}
