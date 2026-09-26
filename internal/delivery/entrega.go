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
	"github.com/go-sob-pressao/enxame/internal/core/policy"
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

// ErrLimite indica que o endpoint chegou ao limite de entregas por
// segundo que o cliente pediu.
var ErrLimite = errors.New("limite de entregas por segundo do endpoint")

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

	mu     sync.Mutex
	vagas  map[string]chan struct{}
	baldes map[string]*policy.TokenBucket
}

// Novo cria o entregador com os padrões do Enxame: breaker de 5 falhas
// seguidas e um minuto de pausa, 4 entregas em curso por endpoint,
// segredos do ambiente.
func Novo(s Store) *Entregador {
	b := &resilience.Breakers{Limiar: 5, Pausa: time.Minute}
	return &Entregador{Store: s, Cliente: NovoCliente(), Breakers: b,
		LimitePorEndpoint: 4, Segredo: SegredoDoAmbiente, Now: time.Now}
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
	if espera, ok := d.taxa(e); !ok {
		return &runner.RepetirEm{Depois: espera, Err: ErrLimite}
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
	status, depois, err := d.enviar(ctx, e, m, &t)
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
	case resultado == webhook.Desacelere && depois > 0:
		// O endpoint disse quando voltar: o pool não repete antes.
		return &runner.RepetirEm{Depois: depois,
			Err: fmt.Errorf("endpoint respondeu %d", status)}
	}
	return fmt.Errorf("endpoint respondeu %d", status)
}

// livro:fim entrega

func (d *Entregador) enviar(
	ctx context.Context,
	e webhook.Endpoint,
	m webhook.Mensagem,
	t *webhook.Tentativa,
) (int, time.Duration, error) {
	segredo, err := d.Segredo(e.SecretRef)
	if err != nil {
		return 0, 0, runner.Permanent(err)
	}
	chave, err := webhook.Chave(segredo)
	if err != nil {
		return 0, 0, runner.Permanent(err)
	}
	corpo, err := json.Marshal(map[string]any{"type": m.EventType,
		"timestamp": m.CriadaEm.UTC(),
		"data":      json.RawMessage(m.Payload)})
	if err != nil {
		return 0, 0, runner.Permanent(err)
	}
	agora := d.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.URL,
		bytes.NewReader(corpo))
	if err != nil {
		return 0, 0, runner.Permanent(err)
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
		return 0, 0, err
	}
	defer resp.Body.Close()
	t.Status, t.Trecho = resp.StatusCode, lerResposta(resp.Body)
	return resp.StatusCode, retryAfter(resp.Header, d.Now()), nil
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

// livro:inicio taxa

// taxa aplica o limite de entregas por segundo do endpoint, se ele
// tiver um. O balde é do processo: com três nós entregando, o endpoint
// recebe até três vezes o limite — o preço de um limite local.
func (d *Entregador) taxa(e webhook.Endpoint) (time.Duration, bool) {
	if e.RateLimit <= 0 {
		return 0, true
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.baldes == nil {
		d.baldes = map[string]*policy.TokenBucket{}
	}
	b, ok := d.baldes[e.ID]
	if !ok || b.Taxa != float64(e.RateLimit) {
		b = &policy.TokenBucket{Taxa: float64(e.RateLimit),
			Rajada: float64(e.RateLimit)}
		d.baldes[e.ID] = b
	}
	ok, espera := b.Tomar(d.Now())
	return espera, ok
}

// livro:fim taxa

// retryAfter lê o cabeçalho Retry-After, em segundos ou como data.
func retryAfter(h http.Header, agora time.Time) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(v); err == nil && s > 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil && t.After(agora) {
		return t.Sub(agora)
	}
	return 0
}
