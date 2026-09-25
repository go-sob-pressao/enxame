// Package agrupador junta eventos de auditoria em lotes antes de
// enviá-los ao armazenamento. É o serviço da Missão #2.
package agrupador

import (
	"context"
	"time"
)

// Agrupador entrega um lote quando ele enche ou quando passa o
// intervalo desde o primeiro evento do lote — o que vier primeiro.
type Agrupador struct {
	tamanho   int
	intervalo time.Duration
	entregar  func(lote []string)
	entrada   chan string
}

// Novo cria um agrupador. Rode Rodar numa goroutine.
func Novo(
	tamanho int,
	intervalo time.Duration,
	entregar func([]string),
) *Agrupador {
	return &Agrupador{
		tamanho:   tamanho,
		intervalo: intervalo,
		entregar:  entregar,
		entrada:   make(chan string),
	}
}

// Adicionar entrega um evento ao agrupador.
func (a *Agrupador) Adicionar(ctx context.Context, ev string) error {
	select {
	case a.entrada <- ev:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Rodar agrupa até o contexto terminar; entrega o que sobrou ao sair.
func (a *Agrupador) Rodar(ctx context.Context) {
	var lote []string
	var prazo <-chan time.Time // nil enquanto o lote está vazio
	entregar := func() {
		a.entregar(lote)
		lote, prazo = nil, nil
	}
	for {
		select {
		case ev := <-a.entrada:
			if len(lote) == 0 {
				prazo = time.After(a.intervalo)
			}
			lote = append(lote, ev)
			if len(lote) == a.tamanho {
				entregar()
			}
		case <-prazo:
			entregar()
		case <-ctx.Done():
			if len(lote) > 0 {
				entregar()
			}
			return
		}
	}
}
