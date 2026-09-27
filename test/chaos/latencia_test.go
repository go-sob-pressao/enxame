//go:build chaos

package chaos

import (
	"context"
	"strings"
	"testing"
	"time"
)

// livro:inicio experimento-latencia

// Experimento 2 — latência entre um nó e o banco.
//
// Hipótese: até 300 ms de latência em cada sentido entre um nó (que
// não é o líder) e o PostgreSQL deixam esse nó mais lento, mas ele não
// perde as partições, e nenhum job se perde.
// Reformulada depois da primeira execução (Cap. 28): "perder" é outro
// nó tomar a partição, não o lease passar um instante do vencimento
// sem ninguém que o dispute — que o experimento mostrou acontecer, e
// mede à parte.
// Injeção: o proxy do nó atrasa cada pedaço em 0 (a referência),
// 100 ms, 300 ms e 1 s, 15 s em cada degrau, com carga.
// Raio de alcance: um nó de três; banco efêmero.
// Refuta: outro nó com a posse de uma partição do nó lento, num degrau
// de até 300 ms; um job perdido.
func TestLatenciaAteOBanco(t *testing.T) {
	l := NovoLaboratorio(t, bin, nil)
	lento := l.Outro(l.Lider())
	entrada := l.Outro(l.Lider(), lento)
	ctx, parar := context.WithCancel(t.Context())
	serie := l.Amostrar(ctx, 500*time.Millisecond)
	fimDaCarga := l.Carga(entrada, 5000, 100)

	l.Aguardar(3 * time.Second)
	refutada := false
	for _, d := range []time.Duration{0, 100 * time.Millisecond,
		300 * time.Millisecond, time.Second} {
		lento.Proxy.Atrasar(d)
		antes := l.ConcluidosPor(lento)
		sem, tomadas, lock, phi, pior := 0, 0, 0, 0.0, "vivo"
		for range 30 {
			l.Aguardar(500 * time.Millisecond)
			if l.Posses()[lento.Nome] < 170 {
				sem++
			}
			tomadas = max(tomadas, l.Tomadas(lento))
			if l.RenovacoesEsperando() > 0 {
				lock++
			}
			e, p, _ := l.Visao(l.Lider(), lento)
			phi = max(phi, p)
			if e != "vivo" && pior != "morto" {
				pior = e
			}
		}
		t.Logf("atraso %v: %s com lease vencido em %d de 30 amostras "+
			"(renovação esperando lock em %d); tomadas por outro nó: "+
			"%d; para o líder, phi até %.1f, pior estado %s; %d jobs "+
			"concluídos pelo worker dele em 15 s", d, lento.Nome, sem,
			lock, tomadas, phi, pior, l.ConcluidosPor(lento)-antes)
		refutada = refutada ||
			(d <= 300*time.Millisecond && tomadas > 0)
	}
	lento.Proxy.Atrasar(0)
	if err := fimDaCarga(); err != nil {
		t.Fatalf("carga: %v", err)
	}
	l.Esperar(3*time.Minute, "todos os jobs concluídos",
		func() bool { return l.Terminados() == 5000 })
	parar()
	saida := l.Conferir(5000)
	t.Log("\n" + saida)
	registrarSerie(t, "latencia", serie())
	if !strings.Contains(saida, "perdidos: 0;") {
		t.Fatal("hipótese refutada: job perdido")
	}
	if refutada {
		t.Fatal("hipótese refutada: outro nó tomou partições com " +
			"até 300 ms")
	}
}

// livro:fim experimento-latencia
