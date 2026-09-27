//go:build chaos

package chaos

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

type relatorio struct{}

func (relatorio) Kind() string { return "relatorio" }

// livro:inicio experimento-relogio

// Experimento 3 — o líder com o relógio adiantado.
//
// Hipótese: um líder com o relógio de parede 3 min adiantado não muda
// quais janelas de um agendamento disparam, nem quando: o Enxame decide
// pelo relógio do banco (Caps. 22 e 24).
// Injeção: ENXAME_CAOS_DESVIO=3m no no-1, que é posto na liderança; um
// agendamento de minuto em minuto; 150 s de observação.
// Raio de alcance: o relógio de um processo; nenhum relógio de máquina
// é tocado.
// Refuta: uma janela disparada antes da hora dela no relógio do banco,
// ou uma janela vencida que não disparou.
func TestRelogioAdiantadoNoLider(t *testing.T) {
	l := NovoLaboratorio(t, bin, func(i int) []string {
		if i == 1 {
			return []string{"ENXAME_CAOS_DESVIO=3m"}
		}
		return nil
	})
	torto := l.Nos[0]
	for l.Lider() != torto { // os outros somem até o no-1 liderar
		for _, n := range l.Nos[1:] {
			n.Proxy.Cortar()
		}
		l.Esperar(20*time.Second, "o no-1 na liderança",
			func() bool { return l.Lider() == torto })
		for _, n := range l.Nos[1:] {
			n.Proxy.Religar()
		}
	}
	var inicio time.Time
	_ = l.DB.QueryRow(t.Context(), `SELECT now()`).Scan(&inicio)
	c := enxame.New(l.DB, "loja")
	if err := c.Schedule(t.Context(), enxame.Schedule{ID: "relatorio",
		Cron: "* * * * *", Queue: "relatorio",
		Args: relatorio{}}); err != nil {
		t.Fatal(err)
	}
	l.Aguardar(150 * time.Second)

	janelas := janelasDisparadas(t, l)
	var fim time.Time
	_ = l.DB.QueryRow(t.Context(), `SELECT now()`).Scan(&fim)
	ultima := fim // inclui as janelas disparadas antes da hora
	for w := range janelas {
		if w.After(ultima) {
			ultima = w
		}
	}
	var relato strings.Builder
	refutada := false
	primeira := inicio.Truncate(time.Minute).Add(time.Minute)
	for w := primeira; !w.After(ultima); w = w.Add(time.Minute) {
		criado, ok := janelas[w.UTC()]
		switch {
		case !ok && (!w.After(fim) || w.Before(ultima)):
			refutada = true
			fmt.Fprintf(&relato, "janela %s: não disparou\n",
				w.Format("15:04"))
		case ok:
			adiantado := criado.Before(w)
			refutada = refutada || adiantado
			fmt.Fprintf(&relato, "janela %s: disparada às %s "+
				"(%+.0f s)\n", w.Format("15:04"),
				criado.Format("15:04:05"),
				criado.Sub(w).Seconds())
		}
	}
	t.Logf("líder %s com o relógio 3 min adiantado:\n%s", torto.Nome,
		relato.String())
	if refutada {
		t.Fatal("hipótese refutada")
	}
}

// livro:fim experimento-relogio

// janelasDisparadas devolve, para cada janela disparada, o instante (no
// relógio do banco) em que o job dela foi criado.
func janelasDisparadas(t *testing.T,
	l *Laboratorio) map[time.Time]time.Time {
	rows, err := l.DB.Query(t.Context(), `SELECT unique_key, created_at
		FROM job WHERE queue = 'relatorio'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	m := map[time.Time]time.Time{}
	for rows.Next() {
		var chave string
		var criado time.Time
		if err := rows.Scan(&chave, &criado); err != nil {
			t.Fatal(err)
		}
		// cron:<namespace>:<id>:<janela RFC 3339>
		partes := strings.SplitN(chave, ":", 4)
		w, err := time.Parse(time.RFC3339, partes[3])
		if err != nil {
			t.Fatal(err)
		}
		m[w.UTC()] = criado
	}
	return m
}
