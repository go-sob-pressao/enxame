// Package avalanche é a Missão #4: a tempestade de retry contra o
// endpoint do cliente que volta de uma queda.
package avalanche

import (
	"slices"
	"time"
)

// Config descreve o cenário.
type Config struct {
	Entregas int // mensagens a entregar, todas de uma vez
	// Queda: o endpoint fica fora do ar do início até aqui.
	Queda      time.Duration
	Capacidade int           // requisições por segundo que ele aguenta
	Limite     time.Duration // até quando a simulação roda
}

// Relatorio é o que a simulação observou.
type Relatorio struct {
	Entregues int
	// Pico: o maior número de requisições num segundo, depois da volta.
	Pico     int
	Recaidas int           // vezes que o endpoint caiu de novo
	UltimaEm time.Duration // quando a última mensagem foi entregue
}

// Proxima é a política de retry sob teste.
type Proxima func(
	n int,
	agora time.Time,
	sorte func() float64,
) time.Time

// Simular roda o cenário, segundo a segundo, num relógio virtual. Uma
// requisição num segundo em que o endpoint está fora do ar falha; um
// segundo com mais requisições que a Capacidade recusa as excedentes
// com 503; um segundo com mais do que o triplo derruba o endpoint por
// mais um minuto — a recaída.
func Simular(c Config, prox Proxima, sorte func() float64) Relatorio {
	t0 := time.Date(2026, 9, 26, 14, 0, 0, 0, time.UTC)
	type pendente struct {
		em time.Time
		n  int
	}
	fila := make([]pendente, c.Entregas)
	for i := range fila {
		fila[i] = pendente{em: t0}
	}
	var r Relatorio
	foraAte := t0.Add(c.Queda)
	for s := time.Duration(0); s < c.Limite; s += time.Second {
		if len(fila) == 0 {
			break
		}
		agora := t0.Add(s)
		fim := agora.Add(time.Second)
		var agoraDevidas, depois []pendente
		for _, p := range fila {
			if p.em.Before(fim) {
				agoraDevidas = append(agoraDevidas, p)
			} else {
				depois = append(depois, p)
			}
		}
		slices.SortFunc(agoraDevidas, func(a, b pendente) int {
			return a.em.Compare(b.em)
		})
		n := len(agoraDevidas)
		if !agora.Before(foraAte) {
			r.Pico = max(r.Pico, n)
			if n > 3*c.Capacidade {
				r.Recaidas++
				foraAte = agora.Add(time.Minute)
			}
		}
		for i, p := range agoraDevidas {
			if !agora.Before(foraAte) && i < c.Capacidade {
				r.Entregues++
				r.UltimaEm = s
				continue
			}
			p.n++
			p.em = prox(p.n, agora, sorte)
			depois = append(depois, p)
		}
		fila = depois
	}
	return r
}
