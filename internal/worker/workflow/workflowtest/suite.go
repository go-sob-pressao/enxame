// Package workflowtest é a suíte que todo armazenamento de runs de
// workflow precisa aprovar, rodando o replay de verdade sobre ele.
package workflowtest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/store"
	wf "github.com/go-sob-pressao/enxame/internal/worker/workflow"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// Store é o armazenamento sob teste: o do replay mais a criação de
// runs.
type Store interface {
	wf.Store
	StartRun(ctx context.Context, run workflow.Run) (string, error)
}

// Run executa a suíte; novo devolve um armazenamento vazio.
func Run(t *testing.T, novo func(t *testing.T) Store) {
	t.Run("PassoConcluidoNaoRodaDeNovo", func(t *testing.T) {
		passoConcluido(t, novo(t))
	})
	t.Run("SleepGuardaAHoraDeAcordar", func(t *testing.T) {
		sleepGuarda(t, novo(t))
	})
	t.Run("NowIgualEmCadaReplay", func(t *testing.T) {
		nowIgual(t, novo(t))
	})
	t.Run("CodigoQueDivergeParaORun", func(t *testing.T) {
		diverge(t, novo(t))
	})
	t.Run("ErroPermanenteEncerraORun", func(t *testing.T) {
		permanente(t, novo(t))
	})
	t.Run("MesmaPosicaoDuasVezes", func(t *testing.T) {
		mesmaPosicao(t, novo(t))
	})
}

func iniciar(t *testing.T, s Store, tipo string) string {
	t.Helper()
	id, err := s.StartRun(t.Context(), workflow.Run{
		Namespace: "ns", WorkflowID: "pedido-42", Type: tipo,
		Input: json.RawMessage(`{"pedido":42}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

type relogio struct{ t time.Time }

func (r *relogio) agora() time.Time { return r.t }

var t0 = time.Date(2026, 9, 25, 14, 0, 0, 0, time.UTC)

func passoConcluido(t *testing.T, s Store) {
	var cobrancas, emails int
	falhaEmail := errors.New("SMTP indisponível")
	fn := func(c *workflow.Context, _ json.RawMessage) (any, error) {
		recibo, err := workflow.Step(c, "cobrar",
			func(context.Context) (string, error) {
				cobrancas++
				return "R-1", nil
			})
		if err != nil {
			return nil, err
		}
		_, err = workflow.Step(c, "enviar-recibo",
			func(context.Context) (bool, error) {
				emails++
				if emails == 1 {
					return false, falhaEmail
				}
				return true, nil
			})
		return recibo, err
	}
	r := &wf.Replayer{Store: s, Funcs: map[string]wf.Func{"f": fn},
		Now: (&relogio{t0}).agora}
	id := iniciar(t, s, "f")
	if _, err := r.Advance(t.Context(), id); !errors.Is(
		err,
		falhaEmail,
	) {
		t.Fatalf("primeira execução: %v", err)
	}
	o, err := r.Advance(t.Context(), id)
	if err != nil || o.State != workflow.RunCompleted {
		t.Fatalf("segunda execução: %+v, %v", o, err)
	}
	if cobrancas != 1 || emails != 2 {
		t.Fatalf("cobranças %d, e-mails %d", cobrancas, emails)
	}
	run, _, _ := s.LoadRun(t.Context(), id)
	if string(run.Output) != `"R-1"` {
		t.Fatalf("saída %s", run.Output)
	}
}

func sleepGuarda(t *testing.T, s Store) {
	rel := &relogio{t0}
	var depois int
	fn := func(c *workflow.Context, _ json.RawMessage) (any, error) {
		if err := workflow.Sleep(c, "esperar", time.Hour); err != nil {
			return nil, err
		}
		depois++
		return nil, nil
	}
	r := &wf.Replayer{Store: s, Funcs: map[string]wf.Func{"f": fn},
		Now: rel.agora}
	id := iniciar(t, s, "f")
	for _, avanco := range []time.Duration{0, 30 * time.Minute} {
		rel.t = t0.Add(avanco)
		o, err := r.Advance(t.Context(), id)
		if err != nil || !o.Until.Equal(t0.Add(time.Hour)) {
			t.Fatalf("aos %v: %+v, %v", avanco, o, err)
		}
	}
	rel.t = t0.Add(time.Hour)
	o, err := r.Advance(t.Context(), id)
	if err != nil || o.State != workflow.RunCompleted || depois != 1 {
		t.Fatalf("depois da hora: %+v, %v, %d", o, err, depois)
	}
}

func nowIgual(t *testing.T, s Store) {
	rel := &relogio{t0}
	var vistos []time.Time
	fn := func(c *workflow.Context, _ json.RawMessage) (any, error) {
		agora, err := workflow.Now(c)
		if err != nil {
			return nil, err
		}
		vistos = append(vistos, agora)
		return nil, workflow.Sleep(c, "esperar", time.Minute)
	}
	r := &wf.Replayer{Store: s, Funcs: map[string]wf.Func{"f": fn},
		Now: rel.agora}
	id := iniciar(t, s, "f")
	for range 3 {
		if _, err := r.Advance(t.Context(), id); err != nil {
			t.Fatal(err)
		}
		rel.t = rel.t.Add(40 * time.Second)
	}
	for _, v := range vistos {
		if !v.Equal(t0) {
			t.Fatalf("Now devolveu %v; esperado %v", vistos, t0)
		}
	}
}

func diverge(t *testing.T, s Store) {
	passo := func(nome string) wf.Func {
		return func(c *workflow.Context, _ json.RawMessage) (any, error) {
			_, err := workflow.Step(c, nome,
				func(context.Context) (int, error) { return 1, nil })
			if err != nil {
				return nil, err
			}
			return nil, workflow.Sleep(c, "esperar", time.Hour)
		}
	}
	id := iniciar(t, s, "f")
	antes := &wf.Replayer{Store: s, Now: (&relogio{t0}).agora,
		Funcs: map[string]wf.Func{"f": passo("reservar-estoque")}}
	if _, err := antes.Advance(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	depois := &wf.Replayer{Store: s, Now: (&relogio{t0}).agora,
		Funcs: map[string]wf.Func{"f": passo("cobrar")}}
	_, err := depois.Advance(t.Context(), id)
	var nd *workflow.NonDeterministicError
	if !errors.As(err, &nd) || nd.Seq != 1 {
		t.Fatalf("esperava NonDeterministicError no passo 1: %v", err)
	}
	run, _, _ := s.LoadRun(t.Context(), id)
	if run.State != workflow.RunRunning {
		t.Fatalf("run %s; deveria continuar aberto", run.State)
	}
}

func permanente(t *testing.T, s Store) {
	var chamadas int
	fn := func(c *workflow.Context, _ json.RawMessage) (any, error) {
		return workflow.Step(c, "cobrar",
			func(context.Context) (int, error) {
				chamadas++
				return 0, workflow.Permanent(
					errors.New("cartão recusado"),
				)
			})
	}
	r := &wf.Replayer{Store: s, Funcs: map[string]wf.Func{"f": fn},
		Now: (&relogio{t0}).agora}
	id := iniciar(t, s, "f")
	for range 2 {
		o, err := r.Advance(t.Context(), id)
		if err != nil || o.State != workflow.RunFailed {
			t.Fatalf("%+v, %v", o, err)
		}
	}
	run, _, _ := s.LoadRun(t.Context(), id)
	if chamadas != 1 || run.Err == "" {
		t.Fatalf("chamadas %d, erro %q", chamadas, run.Err)
	}
}

func mesmaPosicao(t *testing.T, s Store) {
	id := iniciar(t, s, "f")
	r := workflow.Record{Seq: 1, Name: "a", Kind: workflow.KindCall,
		Output: json.RawMessage(`1`)}
	if err := s.AppendStep(t.Context(), id, r); err != nil {
		t.Fatal(err)
	}
	err := s.AppendStep(t.Context(), id, r)
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("segunda gravação: %v", err)
	}
	_, err = s.StartRun(t.Context(), workflow.Run{
		Namespace: "ns", WorkflowID: "pedido-42", Type: "f",
	})
	if !errors.Is(err, store.ErrDuplicate) {
		t.Fatalf("segundo run aberto: %v", err)
	}
}
