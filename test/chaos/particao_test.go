//go:build chaos

package chaos

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// livro:inicio experimento-particao

// Experimento 1 — partição entre o líder e o banco.
//
// Hipótese: isolar o líder do PostgreSQL por 10 s, com carga entrando
// por outro nó, não perde job nem quebra a ordem de nenhuma chave; as
// partições dele voltam a ter dono em menos de 10 s, e quando a rede
// volta ele retoma partições sem intervenção.
// Injeção: o proxy do líder corta todas as conexões por 10 s.
// Raio de alcance: um nó de três; banco efêmero; nada fora do teste.
// Refuta: um job perdido ou fora de ordem; partições do líder sem dono
// por 10 s ou mais; o líder sem partição nenhuma 30 s depois de voltar.
func TestParticaoDoLider(t *testing.T) {
	l := NovoLaboratorio(t, bin, nil)
	lider := l.Lider()
	entrada := l.Outro(lider)
	ctx, parar := context.WithCancel(t.Context())
	serie := l.Amostrar(ctx, 500*time.Millisecond)
	fimDaCarga := l.Carga(entrada, 1000, 100)

	l.Aguardar(3 * time.Second)
	lider.Proxy.Cortar()
	corte := time.Now()
	l.Marcar("corte")
	donos := l.Esperar(30*time.Second, "partições do líder com outro "+
		"dono", func() bool {
		return l.Posses()[lider.Nome] == 0 &&
			soma(l.Posses()) == 512
	})
	l.Marcar("partições com outro dono")
	l.Aguardar(10*time.Second - time.Since(corte))
	lider.Proxy.Religar()
	l.Marcar("religado")
	volta := l.Esperar(60*time.Second, "o líder com partições de novo",
		func() bool { return l.Posses()[lider.Nome] > 0 })

	if err := fimDaCarga(); err != nil {
		t.Fatalf("carga: %v", err)
	}
	ultimo := l.Esperar(90*time.Second, "todos os jobs concluídos",
		func() bool { return l.Terminados() == 1000 })
	parar()
	t.Logf("líder %s isolado 10 s; partições com outro dono em %v; "+
		"de volta com partições %v depois de religado; último job "+
		"%v depois do fim da carga", lider.Nome,
		donos.Round(100*time.Millisecond),
		volta.Round(100*time.Millisecond),
		ultimo.Round(100*time.Millisecond))
	saida := l.Conferir(1000)
	t.Log("\n" + saida)
	registrarSerie(t, "particao", serie())
	escrever(t, "testdata/particao-marcas.csv",
		"t_s,evento\n"+strings.Join(l.marcas, "\n")+"\n")
	if !strings.Contains(saida, "perdidos: 0;") ||
		!strings.Contains(saida, "fora de ordem: 0") {
		t.Fatal("hipótese refutada: job perdido ou fora de ordem")
	}
	if donos >= 10*time.Second {
		t.Fatalf("hipótese refutada: partições sem dono por %v", donos)
	}
}

// livro:fim experimento-particao

func soma(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// registrarSerie grava a série de amostras em testdata/, para a figura.
func registrarSerie(t *testing.T, nome string, as []Amostra) {
	var b strings.Builder
	b.WriteString("t_s,concluidos,no-1,no-2,no-3\n")
	for _, a := range as {
		fmt.Fprintf(&b, "%.1f,%d,%d,%d,%d\n", a.T.Seconds(),
			a.Concluidos, a.Posses["no-1"], a.Posses["no-2"],
			a.Posses["no-3"])
	}
	escrever(t, "testdata/"+nome+".csv", b.String())
}
