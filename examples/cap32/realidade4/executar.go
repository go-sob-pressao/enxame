package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type execucao struct {
	db              *pgxpool.Pool
	api             string
	jobs, workflows int
	duracao, aos    time.Duration
	atualizar       string
	esperar         string
	csv             string
	inicio          time.Time
	mu              sync.Mutex
	tentativas      []tentativa
	eventos         []string
	cli             *http.Client
}

// tentativa é uma requisição à API: quando saiu, quanto levou e o
// status — zero para um erro de rede.
type tentativa struct {
	em       time.Duration
	latencia time.Duration
	status   int
	erro     string
}

func (e *execucao) executar(ctx context.Context) error {
	if err := preparar(ctx, e.db); err != nil {
		return err
	}
	e.cli = &http.Client{Timeout: 10 * time.Second}
	e.inicio = time.Now()
	for i := range e.workflows {
		corpo := map[string]any{"type": "pedido",
			"workflow_id": fmt.Sprintf("r4-%d", i),
			"input":       Pedido{N: i}}
		if err := e.enviar(ctx, "/v1/workflows", corpo); err != nil {
			return err
		}
	}
	e.evento("workflows iniciados")
	amostras := make(chan struct{})
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return e.enfileirar(gctx) })
	g.Go(func() error { return e.atualizarAos(gctx) })
	var linhas []string
	go func() {
		linhas = e.amostrar(ctx, amostras)
	}()
	if err := g.Wait(); err != nil {
		return err
	}
	e.evento("enfileirados e atualizado")
	if err := e.esperarFim(ctx); err != nil {
		return err
	}
	close(amostras)
	<-time.After(600 * time.Millisecond)
	if err := os.WriteFile(e.csv,
		[]byte(strings.Join(linhas, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return e.conferir(ctx)
}

func (e *execucao) evento(s string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	l := fmt.Sprintf("%6.1fs %s", time.Since(e.inicio).Seconds(), s)
	e.eventos = append(e.eventos, l)
	fmt.Println(l)
}

// enviar faz o POST até a API aceitar: um erro de rede ou um 5xx é
// repetido, e um 409 na repetição quer dizer que a primeira chegou.
func (e *execucao) enviar(ctx context.Context, rota string,
	corpo any) error {
	b, err := json.Marshal(corpo)
	if err != nil {
		return err
	}
	for i := range 100 {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			e.api+rota, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer t1")
		req.Header.Set("Content-Type", "application/json")
		t0 := time.Now()
		resp, err := e.cli.Do(req)
		t := tentativa{em: t0.Sub(e.inicio),
			latencia: time.Since(t0)}
		if err == nil {
			t.status = resp.StatusCode
			_ = resp.Body.Close()
		} else {
			t.erro = motivo(err)
		}
		e.mu.Lock()
		e.tentativas = append(e.tentativas, t)
		e.mu.Unlock()
		switch {
		case t.status == http.StatusCreated,
			t.status == http.StatusOK,
			t.status == http.StatusConflict && i > 0:
			return nil
		case t.status >= 400 && t.status < 500 &&
			t.status != http.StatusTooManyRequests:
			return fmt.Errorf("%s: %d", rota, t.status)
		}
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return errors.New(rota + ": a API não aceitou em 100 tentativas")
}

// enfileirar envia os jobs em ritmo constante, cada um com a própria
// chave única: a repetição de um POST não vira um job a mais.
func (e *execucao) enfileirar(ctx context.Context) error {
	passo := e.duracao / time.Duration(e.jobs)
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(64)
	t := time.NewTicker(passo)
	defer t.Stop()
	for i := range e.jobs {
		corpo := map[string]any{"queue": "carga", "kind": "trabalho",
			"unique_key": fmt.Sprintf("r4-%d", i),
			"args":       Trabalho{N: i}}
		g.Go(func() error { return e.enviar(ctx, "/v1/jobs", corpo) })
		select {
		case <-t.C:
		case <-ctx.Done():
			return g.Wait()
		}
	}
	return g.Wait()
}

func (e *execucao) atualizarAos(ctx context.Context) error {
	if e.atualizar == "" {
		return nil
	}
	select {
	case <-time.After(e.aos - time.Since(e.inicio)):
	case <-ctx.Done():
		return ctx.Err()
	}
	e.evento("atualização disparada")
	if err := shell(ctx, e.atualizar); err != nil {
		return err
	}
	if e.esperar != "" {
		if err := shell(ctx, e.esperar); err != nil {
			return err
		}
	}
	e.evento("atualização concluída")
	return nil
}

func shell(ctx context.Context, c string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", c)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	return cmd.Run()
}

type estado struct {
	rodando, concluidos, wfAtivos, wfConcluidos, particoes int
}

func (e *execucao) ler(ctx context.Context) (estado, error) {
	var s estado
	err := e.db.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM job WHERE queue = 'carga'
			AND state = 'running'),
		(SELECT count(*) FROM job WHERE queue = 'carga'
			AND state = 'completed'),
		(SELECT count(*) FROM workflow_run WHERE state = 'running'),
		(SELECT count(*) FROM workflow_run WHERE state = 'completed'),
		(SELECT count(*) FROM partition_lease WHERE owner IS NOT NULL
			AND lease_expires_at > now())`).Scan(&s.rodando,
		&s.concluidos, &s.wfAtivos, &s.wfConcluidos, &s.particoes)
	return s, err
}

// amostrar grava, a cada 500 ms, o estado do banco e o que a API
// respondeu no intervalo.
func (e *execucao) amostrar(ctx context.Context,
	fim <-chan struct{}) []string {
	linhas := []string{"t_s,requisicoes,falhas,p99_ms,rodando," +
		"concluidos,wf_ativos,wf_concluidos,particoes,prontos," +
		"reinicios"}
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	var visto int
	for {
		select {
		case <-t.C:
		case <-fim:
			return linhas
		case <-ctx.Done():
			return linhas
		}
		s, err := e.ler(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "amostra:", err)
			continue
		}
		e.mu.Lock()
		novas := e.tentativas[visto:]
		visto = len(e.tentativas)
		e.mu.Unlock()
		var lat []time.Duration
		falhas := 0
		for _, x := range novas {
			if x.status == 0 || x.status >= 500 {
				falhas++
				continue
			}
			lat = append(lat, x.latencia)
		}
		pr, re := prontos(ctx)
		linhas = append(linhas, fmt.Sprintf(
			"%.1f,%d,%d,%.1f,%d,%d,%d,%d,%d,%d,%d",
			time.Since(e.inicio).Seconds(), len(novas), falhas,
			p99(lat), s.rodando, s.concluidos, s.wfAtivos,
			s.wfConcluidos, s.particoes, pr, re))
	}
}

func p99(xs []time.Duration) float64 {
	if len(xs) == 0 {
		return 0
	}
	slices.Sort(xs)
	return float64(xs[len(xs)*99/100].Microseconds()) / 1000
}

// prontos conta os enxamed prontos e soma os reinícios dos contêineres,
// pela visão do Kubernetes.
func prontos(ctx context.Context) (prontos, reinicios int) {
	out, err := exec.CommandContext(ctx, "kubectl", "get", "pods",
		"-l", "app.kubernetes.io/name=enxamed", "-o",
		"jsonpath={range .items[*]}"+
			`{.status.conditions[?(@.type=="Ready")].status}:`+
			"{.status.containerStatuses[0].restartCount} {end}").
		Output()
	if err != nil {
		return -1, -1
	}
	for p := range strings.FieldsSeq(string(out)) {
		pronto, n, _ := strings.Cut(p, ":")
		if pronto == "True" {
			prontos++
		}
		r, _ := strconv.Atoi(n)
		reinicios += r
	}
	return prontos, reinicios
}

func (e *execucao) esperarFim(ctx context.Context) error {
	limite := time.Now().Add(10 * time.Minute)
	for time.Now().Before(limite) {
		s, err := e.ler(ctx)
		if err == nil && s.concluidos == e.jobs &&
			s.wfConcluidos == e.workflows {
			e.evento("tudo concluído")
			return nil
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return errors.New("não terminou em 10 min")
}

// conferir diz o que se perdeu, o que repetiu e o que a API recusou.
func (e *execucao) conferir(ctx context.Context) error {
	var efeitos, repetidos, passos, passosRep, wfOk int
	err := e.db.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM r4_efeito),
		(SELECT count(*) FROM r4_efeito WHERE execucoes > 1),
		(SELECT count(*) FROM r4_passo),
		(SELECT count(*) FROM r4_passo WHERE execucoes > 1),
		(SELECT count(*) FROM workflow_run WHERE state = 'completed')`).
		Scan(&efeitos, &repetidos, &passos, &passosRep, &wfOk)
	if err != nil {
		return err
	}
	falhas, porque := 0, map[string]int{}
	for _, t := range e.tentativas {
		if t.status == 0 || t.status >= 500 {
			falhas++
			porque[cmp.Or(t.erro, strconv.Itoa(t.status))]++
		}
	}
	fmt.Printf("jobs: %d; com efeito: %d; perdidos: %d; "+
		"rodados mais de uma vez: %d\n", e.jobs, efeitos,
		e.jobs-efeitos, repetidos)
	fmt.Printf("workflows: %d; concluídos: %d; passos: %d de %d; "+
		"passos rodados mais de uma vez: %d\n", e.workflows, wfOk,
		passos, 2*e.workflows, passosRep)
	fmt.Printf("requisições à API: %d; falhas repetidas pelo "+
		"cliente: %d\n", len(e.tentativas), falhas)
	for _, m := range slices.Sorted(maps.Keys(porque)) {
		fmt.Printf("  %6d  %s\n", porque[m], m)
	}
	if efeitos != e.jobs || wfOk != e.workflows ||
		passos != 2*e.workflows {
		return errors.New("REPROVADO: trabalho perdido")
	}
	fmt.Println("APROVADO")
	return nil
}

// motivo resume um erro de rede no que interessa: a última parte.
func motivo(err error) string {
	m := err.Error()
	if i := strings.LastIndex(m, ": "); i >= 0 {
		m = m[i+2:]
	}
	return m
}
