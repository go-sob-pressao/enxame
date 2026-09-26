//go:build integration

package integration_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

type cliente struct {
	t     *testing.T
	base  string
	token string
}

func (c cliente) chamar(metodo, caminho, corpo string) (int, string) {
	c.t.Helper()
	req, err := http.NewRequestWithContext(c.t.Context(), metodo,
		c.base+caminho, strings.NewReader(corpo))
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func apiDeTeste(t *testing.T) (*api.API, *pgxpool.Pool, cliente, cliente) {
	t.Helper()
	db := testutil.Postgres(t)
	a := api.NovaAPI(db, map[string]string{"tk-loja": "loja",
		"tk-outra": "outra"}, slog.New(slog.DiscardHandler))
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	return a, db, cliente{t, srv.URL, "tk-loja"},
		cliente{t, srv.URL, "tk-outra"}
}

func campo(t *testing.T, corpo, nome string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(corpo), &m); err != nil {
		t.Fatalf("%v: %s", err, corpo)
	}
	s, _ := m[nome].(string)
	return s
}

func TestAPIJobs(t *testing.T) {
	_, _, c, outra := apiDeTeste(t)
	if s, _ := (cliente{t, c.base, "errado"}).chamar("GET",
		"/v1/webhooks/endpoints", ""); s != 401 {
		t.Fatalf("sem token: %d", s)
	}
	s, corpo := c.chamar("POST", "/v1/jobs", `{"queue":"q","kind":"eco",
		"args":{"pedido":42},"unique_key":"eco-42"}`)
	if s != 201 {
		t.Fatalf("inserir: %d %s", s, corpo)
	}
	jid := campo(t, corpo, "id")
	if s, _ := outra.chamar("GET", "/v1/jobs/"+jid, ""); s != 404 {
		t.Fatalf("outro namespace viu o job: %d", s)
	}
	if s, corpo := c.chamar("POST", "/v1/jobs", `{"queue":"q",
		"kind":"eco","unique_key":"eco-42"}`); s != 409 {
		t.Fatalf("duplicado: %d %s", s, corpo)
	}
	s, corpo = c.chamar("POST", "/v1/jobs/"+jid+"/cancel", "")
	if s != 200 || campo(t, corpo, "state") != "cancelled" {
		t.Fatalf("cancelar: %d %s", s, corpo)
	}
	if s, corpo := c.chamar("POST", "/v1/jobs/"+jid+"/cancel",
		""); s != 409 {
		t.Fatalf("cancelar de novo: %d %s", s, corpo)
	}
	if s, _ := c.chamar("PUT", "/v1/jobs/"+jid, ""); s != 405 {
		t.Fatalf("método errado: %d", s)
	}
}

// livro:inicio json-entrada

// O json/v2 recusa, por padrão, o que o v1 aceitava em silêncio: um
// campo repetido — qual dos dois vale? — e texto que não é UTF-8.
func TestAPIRecusaJSONAmbiguo(t *testing.T) {
	_, _, c, _ := apiDeTeste(t)
	for nome, corpo := range map[string]string{
		"campo repetido": `{"queue":"q","kind":"eco","kind":"outro"}`,
		"UTF-8 inválido": "{\"queue\":\"q\",\"kind\":\"e\xffco\"}",
		"corpo vazio":    ``,
	} {
		s, resp := c.chamar("POST", "/v1/jobs", corpo)
		t.Logf("%s: %d %s", nome, s, strings.TrimSpace(resp))
		if s != 400 {
			t.Errorf("%s: %d", nome, s)
		}
	}
}

// livro:fim json-entrada

func TestAPIEndpointsEWorkflows(t *testing.T) {
	_, _, c, _ := apiDeTeste(t)
	s, corpo := c.chamar("POST", "/v1/webhooks/endpoints",
		`{"url":"https://cliente.exemplo/hooks",
		  "event_types":["pedido.pago"],"secret_ref":"env:SEGREDO"}`)
	if s != 201 {
		t.Fatalf("criar: %d %s", s, corpo)
	}
	eid := campo(t, corpo, "id")
	if s, corpo := c.chamar("POST", "/v1/webhooks/endpoints",
		`{"url":"ftp://x","secret_ref":"env:S"}`); s != 400 {
		t.Fatalf("url ruim: %d %s", s, corpo)
	}
	s, corpo = c.chamar("GET", "/v1/webhooks/endpoints", "")
	if s != 200 || !strings.Contains(corpo, eid) {
		t.Fatalf("listar: %d %s", s, corpo)
	}
	if s, _ := c.chamar("DELETE", "/v1/webhooks/endpoints/"+eid,
		""); s != 204 {
		t.Fatalf("desativar: %d", s)
	}
	s, corpo = c.chamar("POST", "/v1/workflows",
		`{"type":"pedido","workflow_id":"p-1","input":{"valor":10}}`)
	if s != 201 {
		t.Fatalf("workflow: %d %s", s, corpo)
	}
	s, corpo = c.chamar("GET", "/v1/workflows/"+campo(t, corpo, "id"), "")
	if s != 200 || campo(t, corpo, "state") != "running" {
		t.Fatalf("run: %d %s", s, corpo)
	}
}

// livro:inicio desligamento-teste

// Um cliente espera um job com ?wait=30s. O servidor recebe o sinal de
// desligar: o long-poll responde com o estado atual, e o desligamento
// termina em menos de um segundo depois do aviso — sem esperar os 30.
func TestDesligamentoAcordaLongPoll(t *testing.T) {
	db := testutil.Postgres(t)
	a := api.NovaAPI(db, map[string]string{"tk": "loja"},
		slog.New(slog.DiscardHandler))
	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, desligar := context.WithCancel(t.Context())
	fim := make(chan error, 1)
	go func() {
		fim <- a.Servir(ctx, lis, api.Desligamento{
			Aviso: 200 * time.Millisecond, Prazo: 10 * time.Second})
	}()
	c := cliente{t, "http://" + lis.Addr().String(), "tk"}
	_, corpo := c.chamar("POST", "/v1/jobs",
		`{"queue":"q","kind":"eco"}`)
	jid := campo(t, corpo, "id")
	resposta := make(chan int, 1)
	go func() {
		req, _ := http.NewRequestWithContext(t.Context(),
			http.MethodGet, c.base+"/v1/jobs/"+jid+"?wait=30s", nil)
		req.Header.Set("Authorization", "Bearer tk")
		req.Header.Set("Enxame-Timeout", "40s")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resposta <- -1
			return
		}
		resp.Body.Close()
		resposta <- resp.StatusCode
	}()
	<-time.After(300 * time.Millisecond) // o long-poll está esperando
	inicio := time.Now()
	desligar()
	if err := <-fim; err != nil {
		t.Fatal(err)
	}
	t.Logf("desligou em %v; o long-poll recebeu %d",
		time.Since(inicio).Round(10*time.Millisecond), <-resposta)
	if time.Since(inicio) > time.Second {
		t.Fatal("o desligamento esperou o long-poll")
	}
}

// livro:fim desligamento-teste

// O long-poll volta assim que o job chega a um estado final.
func TestLongPollVoltaNoFim(t *testing.T) {
	a, _, c, _ := apiDeTeste(t)
	_, corpo := c.chamar("POST", "/v1/jobs", `{"queue":"q","kind":"eco"}`)
	jid := campo(t, corpo, "id")
	go func() {
		<-time.After(300 * time.Millisecond)
		id, _ := id.ParseJobID(jid)
		_, _ = a.Store.Decide(context.Background(), id,
			func(j job.Job) ([]job.Event, error) {
				return job.Cancel(j, time.Now())
			})
	}()
	inicio := time.Now()
	s, corpo := c.chamar("GET", "/v1/jobs/"+jid+"?wait=5s", "")
	if s != 200 || campo(t, corpo, "state") != "cancelled" ||
		time.Since(inicio) > 2*time.Second {
		t.Fatalf("%d %s em %v", s, corpo, time.Since(inicio))
	}
}
