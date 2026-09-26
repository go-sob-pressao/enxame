// Package versao é o enigma do Capítulo 17: o passo inserido no meio de
// um workflow, que quebra só os runs em andamento.
package versao

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// Feitos registra os passos executados, para os testes.
type Feitos []string

func (f *Feitos) passo(c *workflow.Context, nome string) error {
	_, err := workflow.Step(c, nome,
		func(context.Context) (bool, error) {
			*f = append(*f, nome)
			return true, nil
		})
	return err
}

// livro:inicio versao-antes

// Antes é o workflow de ontem: cobra, espera a entrega, emite a nota.
func (f *Feitos) Antes(
	c *workflow.Context,
	_ json.RawMessage,
) (any, error) {
	if err := f.passo(c, "cobrar"); err != nil {
		return nil, err
	}
	if err := workflow.Sleep(c, "esperar-entrega",
		48*time.Hour); err != nil {
		return nil, err
	}
	return nil, f.passo(c, "emitir-nota")
}

// livro:fim versao-antes

// livro:inicio versao-depois

// Depois é o deploy de hoje: reserva o estoque entre a cobrança e a
// espera. Correto para pedidos novos.
func (f *Feitos) Depois(
	c *workflow.Context,
	_ json.RawMessage,
) (any, error) {
	if err := f.passo(c, "cobrar"); err != nil {
		return nil, err
	}
	if err := f.passo(c, "reservar-estoque"); err != nil {
		return nil, err
	}
	if err := workflow.Sleep(c, "esperar-entrega",
		48*time.Hour); err != nil {
		return nil, err
	}
	return nil, f.passo(c, "emitir-nota")
}

// livro:fim versao-depois

// livro:inicio versao-correto

// ComVersion é o mesmo deploy, com a mudança marcada no histórico.
func (f *Feitos) ComVersion(
	c *workflow.Context,
	_ json.RawMessage,
) (any, error) {
	if err := f.passo(c, "cobrar"); err != nil {
		return nil, err
	}
	v, err := workflow.Version(c, "reservar-estoque", 0, 1)
	if err != nil {
		return nil, err
	}
	if v == 1 { // runs novos; os antigos seguem sem reservar
		if err := f.passo(c, "reservar-estoque"); err != nil {
			return nil, err
		}
	}
	if err := workflow.Sleep(c, "esperar-entrega",
		48*time.Hour); err != nil {
		return nil, err
	}
	return nil, f.passo(c, "emitir-nota")
}

// livro:fim versao-correto
