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

	fim    chan struct{}  // fechado por Fechar
	fundo  sync.WaitGroup // rotinas de fundo
	fechar sync.Once
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
		fim:        make(chan struct{}),
	}
	n.fundo.Go(func() { n.renovarToken(ctx, renovar) })
	n.fundo.Go(n.publicarMetricas)
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
	// o primeiro sucesso cancela os demais
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// todo envio completa
	resultados := make(chan error, len(n.provedores))
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

// Cancelar remove o assinante e fecha o canal dele: é o fechamento que
// termina o range da goroutine de entrega.
func (n *Notificador) Cancelar(nome string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if c, ok := n.assinantes[nome]; ok {
		close(c)
		delete(n.assinantes, nome)
	}
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
	ctx context.Context,
	renovar func() string,
) {
	tique := time.NewTicker(50 * time.Millisecond)
	defer tique.Stop()
	for {
		select {
		case <-tique.C:
		case <-ctx.Done():
			return
		case <-n.fim:
			return
		}
		t := renovar()
		n.mu.Lock()
		n.token = t
		n.mu.Unlock()
	}
}

func (n *Notificador) publicarMetricas() {
	tique := time.NewTicker(20 * time.Millisecond)
	defer tique.Stop()
	for {
		select {
		case <-tique.C:
		case <-n.fim:
			return
		}
		n.mu.Lock()
		e := n.enviadas
		n.mu.Unlock()
		n.metricas(e)
	}
}

// Fechar encerra o serviço: para as rotinas de fundo, espera por elas e
// encerra as assinaturas restantes.
func (n *Notificador) Fechar() {
	n.fechar.Do(func() {
		close(n.fim)
		n.fundo.Wait()
		n.mu.Lock()
		defer n.mu.Unlock()
		for nome, c := range n.assinantes {
			close(c)
			delete(n.assinantes, nome)
		}
	})
}
