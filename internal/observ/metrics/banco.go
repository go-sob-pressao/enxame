package metrics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// livro:inicio pool

// pool é o USE do pool de conexões: utilização (em uso, do total),
// saturação (quem esperou por uma conexão, e quanto) e — os erros ficam
// nas métricas de cada operação.
type pool struct {
	p                          *pgxpool.Pool
	emUso, ociosas, maximo     *prometheus.Desc
	esperas, espera, aquisicao *prometheus.Desc
}

// livro:fim pool

// RegistrarPool exporta as estatísticas do pool de conexões.
func RegistrarPool(p *pgxpool.Pool) {
	d := func(nome, ajuda string) *prometheus.Desc {
		return prometheus.NewDesc("enxame_db_"+nome, ajuda, nil, nil)
	}
	Registro.MustRegister(&pool{p: p,
		emUso:   d("conexoes_em_uso", "Conexões emprestadas agora."),
		ociosas: d("conexoes_ociosas", "Conexões abertas e livres."),
		maximo:  d("conexoes_maximo", "O tamanho máximo do pool."),
		esperas: d("esperas_total",
			"Aquisições que esperaram uma conexão vagar."),
		espera: d("espera_segundos_total",
			"Tempo somado esperando uma conexão."),
		aquisicao: d("aquisicoes_total", "Conexões emprestadas.")})
}

func (c *pool) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{c.emUso, c.ociosas,
		c.maximo, c.esperas, c.espera, c.aquisicao} {
		ch <- d
	}
}

func (c *pool) Collect(ch chan<- prometheus.Metric) {
	s := c.p.Stat()
	g, k := prometheus.GaugeValue, prometheus.CounterValue
	ch <- prometheus.MustNewConstMetric(c.emUso, g,
		float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.ociosas, g,
		float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.maximo, g,
		float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.esperas, k,
		float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.espera, k,
		s.EmptyAcquireWaitTime().Seconds())
	ch <- prometheus.MustNewConstMetric(c.aquisicao, k,
		float64(s.AcquireCount()))
}

// filas lê do banco, a cada coleta, quantos jobs esperam em cada fila
// e há quanto tempo espera o mais antigo. É a mesma resposta em todos
// os nós; os painéis usam max por fila.
type filas struct {
	db             *pgxpool.Pool
	esperando, ida *prometheus.Desc
}

// RegistrarFilas exporta a profundidade e a idade de cada fila.
func RegistrarFilas(db *pgxpool.Pool) {
	Registro.MustRegister(&filas{db: db,
		esperando: prometheus.NewDesc("enxame_fila_esperando",
			"Jobs disponíveis esperando um worker.",
			[]string{"fila"}, nil),
		ida: prometheus.NewDesc("enxame_fila_mais_antigo_segundos",
			"Há quanto tempo o job disponível mais antigo devia ter "+
				"começado.", []string{"fila"}, nil)})
}

func (c *filas) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.esperando
	ch <- c.ida
}

func (c *filas) Collect(ch chan<- prometheus.Metric) {
	ctx, cancelar := context.WithTimeout(context.Background(),
		2*time.Second)
	defer cancelar()
	rows, err := c.db.Query(ctx, `SELECT queue, count(*),
		extract(epoch FROM now() - min(scheduled_at))
		FROM job WHERE state = 'available' GROUP BY queue`)
	if err != nil {
		return // sem a métrica, e não com um zero que mentiria
	}
	defer rows.Close()
	for rows.Next() {
		var fila string
		var n int64
		var idade float64
		if rows.Scan(&fila, &n, &idade) != nil {
			return
		}
		ch <- prometheus.MustNewConstMetric(c.esperando,
			prometheus.GaugeValue, float64(n), fila)
		ch <- prometheus.MustNewConstMetric(c.ida,
			prometheus.GaugeValue, max(0, idade), fila)
	}
}

// RegistrarCluster exporta o que o nó sabe do cluster: quantas
// partições tem, se é o líder e a época do mapa que segue.
func RegistrarCluster(particoes func() int, lider func() bool,
	epoca func() uint64) {
	Registro.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "enxame_particoes",
			Help: "Partições com posse deste nó."},
			func() float64 { return float64(particoes()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "enxame_lider",
			Help: "1 se este nó é o líder do coordenador."},
			func() float64 {
				if lider() {
					return 1
				}
				return 0
			}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "enxame_mapa_epoca",
			Help: "A época do mapa de partições que o nó segue."},
			func() float64 { return float64(epoca()) }))
}
