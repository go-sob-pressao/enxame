// Package ordem — testes que só passam na ordem em que foram escritos.
package ordem

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

// ErrFechado indica uso do cadastro depois de Fechar.
var ErrFechado = errors.New("cadastro fechado")

// Cadastro guarda clientes em memória, como um repositório de teste.
type Cadastro struct {
	nomes   []string
	fechado bool
}

// Incluir cadastra um cliente novo.
func (c *Cadastro) Incluir(ctx context.Context, nome string) error {
	if err := c.pronto(ctx); err != nil {
		return err
	}
	if slices.Contains(c.nomes, nome) {
		return fmt.Errorf("cliente %q já existe", nome)
	}
	c.nomes = append(c.nomes, nome)
	return nil
}

// Listar devolve os clientes cadastrados.
func (c *Cadastro) Listar(ctx context.Context) ([]string, error) {
	if err := c.pronto(ctx); err != nil {
		return nil, err
	}
	return slices.Clone(c.nomes), nil
}

// Fechar encerra o cadastro; usos posteriores falham.
func (c *Cadastro) Fechar() { c.fechado = true }

func (c *Cadastro) pronto(ctx context.Context) error {
	if c.fechado {
		return ErrFechado
	}
	return ctx.Err()
}
