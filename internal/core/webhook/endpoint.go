package webhook

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalido indica um endpoint que o domínio recusa.
var ErrInvalido = errors.New("endpoint inválido")

// Endpoint é um destino de webhooks inscrito por um cliente.
type Endpoint struct {
	ID          string
	Namespace   string
	URL         string
	Description string
	EventTypes  []string // vazio: todos os tipos
	SecretRef   string   // referência ao segredo; nunca o segredo
	RateLimit   int      // entregas por segundo; zero: sem limite
	Disabled    bool
	CreatedAt   time.Time
}

// Validar confere o endpoint antes de gravá-lo.
func Validar(e Endpoint) error {
	esquema, resto, ok := strings.Cut(e.URL, "://")
	host, _, _ := strings.Cut(resto, "/")
	switch {
	case !ok || host == "" || strings.ContainsAny(e.URL, " \t\n"):
		return fmt.Errorf("%w: url %q", ErrInvalido, e.URL)
	case esquema != "https" && esquema != "http":
		return fmt.Errorf("%w: esquema %q; use https", ErrInvalido,
			esquema)
	case e.SecretRef == "":
		return fmt.Errorf("%w: secret_ref é obrigatório", ErrInvalido)
	case e.RateLimit < 0:
		return fmt.Errorf("%w: rate_limit negativo", ErrInvalido)
	}
	for _, t := range e.EventTypes {
		if strings.TrimSpace(t) == "" {
			return fmt.Errorf("%w: tipo de evento vazio", ErrInvalido)
		}
	}
	return nil
}
