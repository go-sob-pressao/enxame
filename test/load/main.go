// Comando load é o gerador de carga do Capítulo 21: envia POST
// /v1/jobs em malha aberta — no ritmo pedido, responda o servidor
// ou não — e mede a latência a partir do instante em que cada
// requisição deveria ter saído.
//
//	go run ./test/load -token t1 -fases 500:10s,5000:20s,500:20s
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sob-pressao/enxame/test/load/carga"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "base da API")
	token := flag.String("token", "", "token do namespace")
	fases := flag.String("fases", "500:10s",
		"taxa:duração,… em sequência")
	emVoo := flag.Int("max-em-voo", 20000,
		"requisições abertas no gerador")
	conexoes := flag.Int("conexoes", 512, "conexões TCP com a API")
	csv := flag.String("csv", "", "linha do tempo por segundo")
	brutos := flag.String("brutos", "", "uma linha por requisição")
	tamanho := flag.Int("bytes", 0, "bytes de dados nos argumentos")
	flag.Parse()
	fs, err := lerFases(*fases)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	g := carga.Gerador{URL: *url + "/v1/jobs", Token: *token,
		MaxEmVoo: *emVoo, Conexoes: *conexoes}
	if *tamanho > 0 {
		g.Corpo = carga.CorpoCom(*tamanho)
	}
	rs := g.Executar(context.Background(), fs)
	carga.Resumir(os.Stdout, rs)
	if *csv != "" {
		must(gravar(*csv, func(w io.Writer) {
			carga.LinhaDoTempo(w, rs)
		}))
	}
	if *brutos != "" {
		must(gravar(*brutos, func(w io.Writer) {
			fmt.Fprintln(w, "intencao_ms,latencia_ms,status")
			for _, r := range rs {
				fmt.Fprintf(w, "%.1f,%.3f,%d\n", carga.Ms(r.Intencao),
					carga.Ms(r.Latencia), r.Status)
			}
		}))
	}
}

func lerFases(s string) ([]carga.Fase, error) {
	var fs []carga.Fase
	for p := range strings.SplitSeq(s, ",") {
		taxa, dur, ok := strings.Cut(p, ":")
		t, err1 := strconv.ParseFloat(taxa, 64)
		d, err2 := time.ParseDuration(dur)
		if !ok || err1 != nil || err2 != nil || t <= 0 || d <= 0 {
			return nil, fmt.Errorf("fase inválida: %q", p)
		}
		fs = append(fs, carga.Fase{Taxa: t, Duracao: d})
	}
	if len(fs) == 0 {
		return nil, errors.New("nenhuma fase")
	}
	return fs, nil
}

func gravar(caminho string, f func(io.Writer)) error {
	a, err := os.Create(caminho)
	if err != nil {
		return err
	}
	f(a)
	return a.Close()
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
