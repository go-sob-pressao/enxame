// Package notificador envia notificações por vários provedores, entrega
// eventos a assinantes e publica métricas. É o serviço da Missão #1.
package notificador

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Provedor envia uma mensagem (SMS, e-mail, push…).
type Provedor func(ctx context.Context, destino, msg string) error

// Notificador é o serviço.
type Notificador struct {
	provedores []Provedor
	metricas   func(enviadas int)

	mu         sync.Mutex
	enviadas   int
	assinantes map[string]chan string
	token      string
}

// Novo cria o serviço e inicia as rotinas de fundo.
func Novo(
	ctx context.Context,
	provedores []Provedor,
	renovar func() string,
	metricas func(int),
) *Notificador {
	n := &Notificador{
		provedores: provedores,
		metricas:   metricas,
		assinantes: map[string]chan string{},
		token:      renovar(),
	}
	go n.renovarToken(ctx, renovar)
	go n.publicarMetricas()
	return n
}

// ErrSemProvedor indica que nenhum provedor aceitou a mensagem.
var ErrSemProvedor = errors.New("nenhum provedor disponível")

// Enviar tenta todos os provedores ao mesmo tempo e fica com o primeiro
// que responder com sucesso.
func (n *Notificador) Enviar(
	ctx context.Context,
	destino, msg string,
) error {
	resultados := make(chan error)
	for _, p := range n.provedores {
		go func() { resultados <- p(ctx, destino, msg) }()
	}
	for range n.provedores {
		if err := <-resultados; err == nil {
			n.mu.Lock()
			n.enviadas++
			n.mu.Unlock()
			return nil
		}
	}
	return ErrSemProvedor
}

// Assinar registra um assinante e entrega a ele cada evento publicado.
func (n *Notificador) Assinar(nome string, receber func(string)) {
	c := make(chan string, 16)
	n.mu.Lock()
	n.assinantes[nome] = c
	n.mu.Unlock()
	go func() {
		for evento := range c {
			receber(evento)
		}
	}()
}

// Cancelar remove o assinante.
func (n *Notificador) Cancelar(nome string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.assinantes, nome)
}

// Publicar entrega um evento a todos os assinantes.
func (n *Notificador) Publicar(evento string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, c := range n.assinantes {
		c <- evento
	}
}

// Token devolve o token corrente do provedor.
func (n *Notificador) Token() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.token
}

func (n *Notificador) renovarToken(
	_ context.Context,
	renovar func() string,
) {
	for {
		<-time.After(50 * time.Millisecond)
		t := renovar()
		n.mu.Lock()
		n.token = t
		n.mu.Unlock()
	}
}

func (n *Notificador) publicarMetricas() {
	for range time.NewTicker(20 * time.Millisecond).C {
		n.mu.Lock()
		e := n.enviadas
		n.mu.Unlock()
		n.metricas(e)
	}
}

// Fechar encerra o serviço.
func (n *Notificador) Fechar() {}
