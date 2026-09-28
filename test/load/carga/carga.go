// Package carga é o gerador de carga em malha aberta do Capítulo 21,
// usado pelo comando test/load e pelo Teste de Realidade #2.
package carga

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// Fase é um trecho de carga constante.
type Fase struct {
	Taxa    float64 // requisições por segundo
	Duracao time.Duration
}

// Resultado é uma requisição medida.
type Resultado struct {
	Intencao time.Duration // quando deveria sair, desde o início
	Latencia time.Duration // da intenção até a resposta
	Status   int           // 0: erro de rede ou prazo do cliente
}

// NaoEnviada marca a requisição que o gerador não chegou a enviar.
const NaoEnviada = -1

// livro:inicio malha-aberta

// Gerador envia em malha aberta. A requisição i tem hora marcada; se o
// gerador se atrasa ou o servidor demora, a espera entra na latência.
// Um gerador em malha fechada — manda a próxima quando a anterior
// volta — desacelera junto com o servidor e mede a latência de um
// sistema que não está sob a carga que se queria testar: é a omissão
// coordenada.
type Gerador struct {
	URL, Token string
	MaxEmVoo   int
	Conexoes   int
	Corpo      []byte // nil: um job de argumentos vazios
	cliente    *http.Client
}

// Executar roda as fases em sequência e devolve cada requisição.
func (g *Gerador) Executar(ctx context.Context, fs []Fase) []Resultado {
	// Conexões reaproveitadas, como num cliente de verdade: sem teto,
	// o gerador esgota as portas efêmeras antes de o servidor cair.
	g.cliente = &http.Client{Timeout: 30 * time.Second,
		Transport: &http.Transport{MaxConnsPerHost: g.Conexoes,
			MaxIdleConnsPerHost: g.Conexoes}}
	var (
		mu  sync.Mutex
		rs  []Resultado
		wg  sync.WaitGroup
		voo = make(chan struct{}, g.MaxEmVoo)
	)
	inicio, base := time.Now(), time.Duration(0)
	for _, f := range fs {
		passo := time.Duration(float64(time.Second) / f.Taxa)
		for t := time.Duration(0); t < f.Duracao; t += passo {
			intencao := base + t
			if d := time.Until(inicio.Add(intencao)); d > 0 {
				time.Sleep(d)
			}
			select {
			case voo <- struct{}{}:
			default: // o gerador também tem limite: conta como falha
				mu.Lock()
				rs = append(rs, Resultado{intencao, 0, NaoEnviada})
				mu.Unlock()
				continue
			}
			wg.Go(func() {
				defer func() { <-voo }()
				st := g.enviar(ctx)
				r := Resultado{intencao,
					time.Since(inicio) - intencao, st}
				mu.Lock()
				rs = append(rs, r)
				mu.Unlock()
			})
		}
		base += f.Duracao
	}
	wg.Wait()
	slices.SortFunc(rs, func(a, b Resultado) int {
		return cmp.Compare(a.Intencao, b.Intencao)
	})
	return rs
}

// livro:fim malha-aberta

var corpo = []byte(`{"queue":"carga","kind":"eco","args":{}}`)

func (g *Gerador) corpo() []byte {
	if g.Corpo == nil {
		return corpo
	}
	return g.Corpo
}

// CorpoCom devolve um job cujos argumentos têm n bytes de dados.
func CorpoCom(n int) []byte {
	return []byte(`{"queue":"carga","kind":"eco","args":{"dados":"` +
		strings.Repeat("x", n) + `"}}`)
}

func (g *Gerador) enviar(ctx context.Context) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.URL,
		bytes.NewReader(g.corpo()))
	if err != nil {
		return 0
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.cliente.Do(req)
	if err != nil {
		return 0
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

// Percentil devolve o percentil p (0–100) de latências ordenadas.
func Percentil(ls []time.Duration, p float64) time.Duration {
	if len(ls) == 0 {
		return 0
	}
	i := int(float64(len(ls)-1) * p / 100)
	return ls[i]
}

// Aceitas devolve as latências das respostas 2xx, ordenadas.
func Aceitas(rs []Resultado) []time.Duration {
	var ls []time.Duration
	for _, r := range rs {
		if r.Status/100 == 2 {
			ls = append(ls, r.Latencia)
		}
	}
	slices.Sort(ls)
	return ls
}

// Resumir escreve o total por status e os percentis das aceitas.
func Resumir(w io.Writer, rs []Resultado) {
	porStatus := map[int]int{}
	for _, r := range rs {
		porStatus[r.Status]++
	}
	ls := Aceitas(rs)
	fmt.Fprintf(w, "enviadas %d  status %v\n", len(rs), porStatus)
	fmt.Fprintf(w, "aceitas: p50 %v  p99 %v  p99.9 %v  máx %v\n",
		Percentil(ls, 50).Round(time.Microsecond*100),
		Percentil(ls, 99).Round(time.Microsecond*100),
		Percentil(ls, 99.9).Round(time.Microsecond*100),
		Percentil(ls, 100).Round(time.Microsecond*100))
}

// LinhaDoTempo agrupa por segundo de intenção: taxa enviada, aceitas,
// recusadas e a latência das aceitas.
func LinhaDoTempo(w io.Writer, rs []Resultado) {
	fmt.Fprintln(w, "segundo,enviadas,aceitas,r429,r503,falhas,"+
		"p50_ms,p99_ms")
	for s := 0; len(rs) > 0; s++ {
		fim := time.Duration(s+1) * time.Second
		n, _ := slices.BinarySearchFunc(rs, fim,
			func(r Resultado, t time.Duration) int {
				return cmp.Compare(r.Intencao, t)
			})
		seg := rs[:n]
		rs = rs[n:]
		ls := Aceitas(seg)
		c := map[int]int{}
		for _, r := range seg {
			c[r.Status]++
		}
		falhas := len(seg) - len(ls) - c[429] - c[503]
		fmt.Fprintf(w, "%d,%d,%d,%d,%d,%d,%.2f,%.2f\n", s, len(seg),
			len(ls), c[429], c[503], falhas,
			Ms(Percentil(ls, 50)), Ms(Percentil(ls, 99)))
	}
}

// Ms converte para milissegundos.
func Ms(d time.Duration) float64 { return float64(d) / 1e6 }
