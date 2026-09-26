package membership

import (
	"math"
	"time"
)

// Estado é o que um detector diz de um nó.
type Estado int

// Os três estados. Suspeito não é meio morto: é "ainda não sei".
const (
	Vivo Estado = iota
	Suspeito
	Morto
)

func (e Estado) String() string {
	return [...]string{"vivo", "suspeito", "morto"}[e]
}

// Detector recebe as batidas de um nó e diz, a cada instante, o que
// acha dele. Os instantes vêm de fora, de um relógio só.
type Detector interface {
	Batida(em time.Time)
	Estado(agora time.Time) Estado
}

// livro:inicio fixo

// Fixo é o detector de timeout fixo: morto depois de Limite sem batida.
// Não tem suspeita, e não aprende nada com o que já viu.
type Fixo struct {
	Limite time.Duration
	ultima time.Time
}

// Batida registra uma batida.
func (f *Fixo) Batida(em time.Time) { f.ultima = em }

// Estado diz vivo ou morto.
func (f *Fixo) Estado(agora time.Time) Estado {
	if agora.Sub(f.ultima) > f.Limite {
		return Morto
	}
	return Vivo
}

// livro:fim fixo

// livro:inicio phi

// Phi é o detector phi accrual: guarda os últimos intervalos entre
// batidas e, a cada instante, calcula quão improvável é o silêncio
// atual se o nó estivesse vivo. phi = −log10 da probabilidade de a
// próxima batida chegar ainda mais tarde: phi 1 é uma chance em 10 de
// o nó estar vivo e só atrasado; phi 8, uma em 100 milhões.
type Phi struct {
	Janela   int           // quantos intervalos lembrar (1000)
	Suspeita float64       // phi a partir do qual o nó é suspeito
	Morte    float64       // phi a partir do qual o nó é dado morto
	Folga    time.Duration // pausa aceitável somada à média
	Desvio   time.Duration // desvio mínimo, contra a precisão falsa
	Esperado time.Duration // intervalo suposto antes de haver história

	intervalos []time.Duration
	ultima     time.Time
}

// Batida registra uma batida e o intervalo desde a anterior.
func (p *Phi) Batida(em time.Time) {
	if !p.ultima.IsZero() && em.After(p.ultima) {
		p.intervalos = append(p.intervalos, em.Sub(p.ultima))
		if len(p.intervalos) > p.Janela {
			p.intervalos = p.intervalos[1:]
		}
	}
	p.ultima = em
}

// Valor calcula phi para o silêncio desde a última batida, supondo que
// os intervalos seguem uma distribuição normal com a média e o desvio
// observados.
func (p *Phi) Valor(agora time.Time) float64 {
	if p.ultima.IsZero() {
		return 0 // nunca bateu: não há de onde contar o silêncio
	}
	hist := p.intervalos
	if len(hist) == 0 {
		// Sem história, supõe o intervalo esperado, com um quarto de
		// desvio para cada lado: quem morre na primeira batida também
		// precisa ser dado por morto.
		e := p.Esperado
		hist = []time.Duration{e - e/4, e + e/4}
	}
	var soma, quad float64
	for _, i := range hist {
		s := i.Seconds()
		soma += s
		quad += s * s
	}
	n := float64(len(hist))
	media := soma/n + p.Folga.Seconds()
	desvio := max(math.Sqrt(max(quad/n-(soma/n)*(soma/n), 0)),
		p.Desvio.Seconds())
	y := (agora.Sub(p.ultima).Seconds() - media) / desvio
	depois := 0.5 * math.Erfc(y/math.Sqrt2) // P(chegar mais tarde)
	if depois < 1e-300 {
		return 300 // o limite do float64: certeza, na prática
	}
	return max(-math.Log10(depois), 0) // sem o −0 de log10(1)
}

// Estado traduz phi nos três estados.
func (p *Phi) Estado(agora time.Time) Estado {
	switch v := p.Valor(agora); {
	case v >= p.Morte:
		return Morto
	case v >= p.Suspeita:
		return Suspeito
	}
	return Vivo
}

// livro:fim phi

// NovoPhi cria o detector com os padrões do Enxame para batidas a cada
// esperado: janela de 1000 intervalos, suspeita em phi 3, morte em phi
// 8, desvio mínimo de 100 ms e folga de 3 s. A folga é a pausa que se
// aceita sem desconfiar, e é ela, não a janela, que absorve as pausas
// raras de coleta de lixo (Experimento 23.1).
func NovoPhi(esperado time.Duration) *Phi {
	return &Phi{Janela: 1000, Suspeita: 3, Morte: 8,
		Folga: 3 * time.Second, Desvio: 100 * time.Millisecond,
		Esperado: esperado}
}
