package notificador

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.uber.org/goleak"
)

func ok(context.Context, string, string) error { return nil }

func ruim(
	context.Context,
	string,
	string,
) error {
	return errors.New("fora do ar")
}

func servico(ctx context.Context) *Notificador {
	return Novo(
		ctx,
		[]Provedor{ok, ruim, ok},
		func() string { return "tok" },
		func(int) {},
	)
}

func TestEnviarUsaOPrimeiroQueResponde(t *testing.T) {
	ctx := t.Context()
	n := servico(ctx)
	defer n.Fechar()
	if err := n.Enviar(ctx, "+5511999990000", "pedido enviado"); err != nil {
		t.Fatal(err)
	}
}

func TestAssinantesRecebemEventos(t *testing.T) {
	n := servico(t.Context())
	defer n.Fechar()
	var wg sync.WaitGroup
	wg.Add(2)
	n.Assinar("a", func(string) { wg.Done() })
	n.Assinar("b", func(string) { wg.Done() })
	n.Publicar("pedido.criado")
	wg.Wait()
	n.Cancelar("a")
	n.Cancelar("b")
}

// livro:inicio missao-01-teste

// TestMissao usa o serviço como a aplicação usa — sobe, envia, assina,
// cancela a assinatura, fecha — e exige que nada fique para trás.
func TestMissao(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	n := servico(ctx)
	for range 10 {
		_ = n.Enviar(ctx, "+5511999990000", "oi")
	}
	n.Assinar("painel", func(string) {})
	n.Publicar("pedido.criado")
	n.Cancelar("painel")
	n.Fechar()
	cancel()
}

// livro:fim missao-01-teste
