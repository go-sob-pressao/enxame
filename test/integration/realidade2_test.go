//go:build integration

package integration_test

import (
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/test/load/carga"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio realidade2

// Teste de Realidade #2: carga normal, uma avalanche bem acima da
// capacidade, carga normal de novo. Durante a avalanche, o Enxame
// recusa o excesso e quem entra tem latência limitada; depois dela,
// volta ao normal sem fila para drenar; e todo job aceito existe.
func TestRealidade2(t *testing.T) {
	if comRace {
		// O gerador roda no mesmo processo, e com -race ele se atrasa
		// tanto que mede a si mesmo, e não o servidor.
		t.Skip("teste de carga: rode sem -race")
	}
	db := testutil.Postgres(t)
	a := api.NovaAPI(db, map[string]string{"t": "carga"},
		slog.New(slog.DiscardHandler))
	a.MaxEmCurso = 32 // o padrão do enxamed
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()

	normal, avalanche := 200.0, 5000.0
	g := carga.Gerador{URL: srv.URL + "/v1/jobs", Token: "t",
		MaxEmVoo: 20000, Conexoes: 256}
	rs := g.Executar(t.Context(), []carga.Fase{
		{Taxa: normal, Duracao: 3 * time.Second},
		{Taxa: avalanche, Duracao: 5 * time.Second},
		{Taxa: normal, Duracao: 4 * time.Second},
	})
	antes := trecho(rs, 0, 3*time.Second)
	durante := trecho(rs, 3*time.Second, 8*time.Second)
	depois := trecho(rs, 10*time.Second, 12*time.Second)
	base := p99(antes)
	t.Logf("antes: p99 %v; durante: %d recusadas, p99 %v; "+
		"depois: p99 %v", base, recusadas(durante), p99(durante),
		p99(depois))

	if p99(durante) > time.Second {
		t.Errorf("p99 das aceitas na avalanche: %v", p99(durante))
	}
	if recusadas(durante) == 0 {
		t.Error("nada recusado: a avalanche não passou da " +
			"capacidade, ou nada recusou o excesso")
	}
	// Recuperar é voltar ao normal: no máximo 1% recusado — um pico
	// passageiro ainda pode encostar no limite — e o p99 de antes.
	if r := recusadas(depois); r > len(depois)/100 ||
		p99(depois) > max(5*base, 200*time.Millisecond) {
		t.Errorf("sem recuperação: %d recusadas, p99 %v", r,
			p99(depois))
	}
	aceitas, falhas := 0, 0
	for _, r := range rs {
		switch {
		case r.Status == 201:
			aceitas++
		case r.Status != 503:
			falhas++
		}
	}
	if falhas > 0 {
		t.Errorf("%d requisições sem resposta 201 ou 503", falhas)
	}
	if n := contar(t, db, `SELECT count(*) FROM job`); n != aceitas {
		t.Errorf("%d jobs no banco para %d respostas 201", n, aceitas)
	}
}

// livro:fim realidade2

func trecho(
	rs []carga.Resultado,
	de, ate time.Duration,
) []carga.Resultado {
	var s []carga.Resultado
	for _, r := range rs {
		if r.Intencao >= de && r.Intencao < ate {
			s = append(s, r)
		}
	}
	return s
}

func p99(rs []carga.Resultado) time.Duration {
	return carga.Percentil(carga.Aceitas(rs), 99)
}

func recusadas(rs []carga.Resultado) int {
	n := 0
	for _, r := range rs {
		if r.Status == 503 {
			n++
		}
	}
	return n
}
