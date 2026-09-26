package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/transport/resilience"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

// Store é o que a entrega precisa do armazenamento.
type Store interface {
	Mensagem(ctx context.Context, id string) (webhook.Mensagem, error)
	Endpoint(ctx context.Context, id string) (webhook.Endpoint, error)
	ListEndpoints(ctx context.Context, namespace string) (
		[]webhook.Endpoint, error)
	EnfileirarEntregas(ctx context.Context, m webhook.Mensagem,
		endpoints []string, args func(string) []byte) (int, error)
	RegistrarTentativa(ctx context.Context, t webhook.Tentativa) error
	DisableEndpoint(ctx context.Context, namespace, id,
		motivo string) error
}

// ErrOcupado indica que o endpoint já tem entregas demais em curso.
var ErrOcupado = errors.New("endpoint com entregas demais em curso")

// Entregador executa os jobs de fan-out e de entrega.
type Entregador struct {
	Store    Store
	Cliente  *http.Client
	Breakers *resilience.Breakers
	// LimitePorEndpoint é o bulkhead: quantas entregas a um mesmo
	// endpoint podem estar em curso ao mesmo tempo.
	LimitePorEndpoint int
	// Segredo resolve a referência do endpoint no segredo whsec_…
	Segredo func(ref string) (string, error)
	Now     func() time.Time

	mu    sync.Mutex
	vagas map[string]chan struct{}
}

type argsEntrega struct {
	MessageID  string `json:"message_id"`
	EndpointID string `json:"endpoint_id"`
}

// Fanout cria um job de entrega por endpoint inscrito no tipo.
func (d *Entregador) Fanout(ctx context.Context, j job.Job) error {
	var a struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(j.Args, &a); err != nil {
		return runner.Permanent(err)
	}
	m, err := d.Store.Mensagem(ctx, a.MessageID)
	if err != nil {
		return err
	}
	eps, err := d.Store.ListEndpoints(ctx, m.Namespace)
	if err != nil {
		return err
	}
	var destinos []string
	for _, e := range eps {
		if webhook.Inscrito(e, m.EventType) {
			destinos = append(destinos, e.ID)
		}
	}
	_, err = d.Store.EnfileirarEntregas(ctx, m, destinos,
		func(e string) []byte {
			b, _ := json.Marshal(argsEntrega{m.ID, e})
			return b
		})
	return err
}

// livro:inicio entrega

// Entregar faz uma tentativa de entrega de uma mensagem a um endpoint:
// pergunta ao breaker, pega uma vaga no bulkhead, assina, envia com
// prazo, registra a tentativa — sempre, com ou sem resposta — e decide
// pelo status. 410 desativa o endpoint; o resto que não é 2xx volta à
// fila, com backoff.
func (d *Entregador) Entregar(ctx context.Context, j job.Job) error {
	var a argsEntrega
	if err := json.Unmarshal(j.Args, &a); err != nil {
		return runner.Permanent(err)
	}
	e, err := d.Store.Endpoint(ctx, a.EndpointID)
	if err != nil || e.Disabled {
		return err // desativado: nada a entregar
	}
	if err := d.Breakers.Permitir(e.ID, d.Now()); err != nil {
		return err
	}
	liberar, err := d.vaga(e.ID)
	if err != nil {
		return err
	}
	defer liberar()
	m, err := d.Store.Mensagem(ctx, a.MessageID)
	if err != nil {
		return err
	}
	t := webhook.Tentativa{MessageID: m.ID, EndpointID: e.ID,
		Attempt: j.Attempt, JobID: j.ID.String(), Em: d.Now()}
	status, err := d.enviar(ctx, e, m, &t)
	if err != nil {
		t.Erro = err.Error()
	}
	if errReg := d.Store.RegistrarTentativa(ctx, t); errReg != nil {
		return errReg
	}
	resultado := webhook.Classificar(status)
	d.Breakers.Registrar(e.ID, d.Now(),
		err == nil && resultado == webhook.Entregue)
	switch {
	case err != nil:
		return err
	case resultado == webhook.Entregue:
		return nil
	case resultado == webhook.Desative:
		motivo := "o endpoint respondeu 410 Gone"
		return runner.Permanent(errors.Join(errors.New(motivo),
			d.Store.DisableEndpoint(ctx, e.Namespace, e.ID, motivo)))
	}
	return fmt.Errorf("endpoint respondeu %d", status)
}

// livro:fim entrega

func (d *Entregador) enviar(
	ctx context.Context,
	e webhook.Endpoint,
	m webhook.Mensagem,
	t *webhook.Tentativa,
) (int, error) {
	segredo, err := d.Segredo(e.SecretRef)
	if err != nil {
		return 0, runner.Permanent(err)
	}
	chave, err := webhook.Chave(segredo)
	if err != nil {
		return 0, runner.Permanent(err)
	}
	corpo, err := json.Marshal(map[string]any{"type": m.EventType,
		"timestamp": m.CriadaEm.UTC(),
		"data":      json.RawMessage(m.Payload)})
	if err != nil {
		return 0, runner.Permanent(err)
	}
	agora := d.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.URL,
		bytes.NewReader(corpo))
	if err != nil {
		return 0, runner.Permanent(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("webhook-id", m.ID)
	req.Header.Set("webhook-timestamp",
		strconv.FormatInt(agora.Unix(), 10))
	req.Header.Set("webhook-signature",
		webhook.Assinar(chave, m.ID, agora, corpo))
	resp, err := d.Cliente.Do(req)
	t.Duracao = d.Now().Sub(agora)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	t.Status, t.Trecho = resp.StatusCode, lerResposta(resp.Body)
	return resp.StatusCode, nil
}

// vaga ocupa uma das vagas do endpoint, sem esperar: sem vaga, a
// entrega volta à fila, em vez de prender um worker atrás de um
// endpoint lento.
func (d *Entregador) vaga(endpoint string) (func(), error) {
	d.mu.Lock()
	if d.vagas == nil {
		d.vagas = map[string]chan struct{}{}
	}
	c, ok := d.vagas[endpoint]
	if !ok {
		c = make(chan struct{}, max(d.LimitePorEndpoint, 1))
		d.vagas[endpoint] = c
	}
	d.mu.Unlock()
	select {
	case c <- struct{}{}:
		return func() { <-c }, nil
	default:
		return nil, ErrOcupado
	}
}

// SegredoDoAmbiente resolve referências env:NOME.
func SegredoDoAmbiente(ref string) (string, error) {
	nome, ok := strings.CutPrefix(ref, "env:")
	if !ok {
		return "", fmt.Errorf("referência de segredo %q: só env:NOME "+
			"é aceita", ref)
	}
	v := os.Getenv(nome)
	if v == "" {
		return "", fmt.Errorf("variável %s vazia", nome)
	}
	return v, nil
}
