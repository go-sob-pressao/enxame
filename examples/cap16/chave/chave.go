// Package chave é o enigma do Capítulo 16: a chave de idempotência
// gerada a cada tentativa.
package chave

import (
	"context"
	"encoding/json"
	"uuid"

	"github.com/go-sob-pressao/enxame/examples/cap16/gateway"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	pkgjob "github.com/go-sob-pressao/enxame/pkg/job"
)

// Assinatura são os argumentos do job de cobrança mensal.
type Assinatura struct {
	Cliente string `json:"cliente"`
	Valor   int    `json:"valor"`
}

// livro:inicio chave-defeito

// CobrarComUUID usa uma chave de idempotência — nova a cada chamada.
func CobrarComUUID(g *gateway.Gateway) func(context.Context, job.Job) error {
	return func(ctx context.Context, j job.Job) error {
		var a Assinatura
		if err := json.Unmarshal(j.Args, &a); err != nil {
			return err
		}
		chave := uuid.NewV7().String() // "idempotência"
		_, err := g.Cobrar(ctx, chave, a.Cliente, a.Valor)
		return err
	}
}

// livro:fim chave-defeito

// livro:inicio chave-correta

// Cobrar usa a chave do job, a mesma em todas as tentativas.
func Cobrar(g *gateway.Gateway) func(context.Context, job.Job) error {
	return func(ctx context.Context, j job.Job) error {
		var a Assinatura
		if err := json.Unmarshal(j.Args, &a); err != nil {
			return err
		}
		chave := pkgjob.IdempotencyKey(ctx)
		_, err := g.Cobrar(ctx, chave, a.Cliente, a.Valor)
		return err
	}
}

// livro:fim chave-correta
