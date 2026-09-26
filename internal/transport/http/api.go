package http

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

// API é a API pública do Enxame.
type API struct {
	DB     *pgxpool.Pool
	Store  *postgres.Store
	Tokens map[string]string // token → namespace
	Log    *slog.Logger
	// Taxa e Rajada limitam cada namespace; MaxEmCurso limita as
	// requisições em curso no processo. Zero desliga cada um.
	Taxa, Rajada float64
	MaxEmCurso   int
	// encerrando é fechado quando o servidor começa a desligar: os
	// long-polls param de esperar.
	encerrando chan struct{}
	parando    atomic.Bool
}

// NovaAPI cria a API.
func NovaAPI(
	db *pgxpool.Pool,
	tokens map[string]string,
	log *slog.Logger,
) *API {
	return &API{DB: db, Store: postgres.New(db), Tokens: tokens,
		Log: log, encerrando: make(chan struct{})}
}

// Erro é o corpo de toda resposta de erro.
type Erro struct {
	Codigo   string `json:"code"`
	Mensagem string `json:"message"`
}

// erroHTTP associa erros de domínio a status HTTP.
var erroHTTP = []struct {
	erro   error
	status int
	codigo string
}{
	{store.ErrNotFound, http.StatusNotFound, "not_found"},
	{
		job.ErrInvalidTransition,
		http.StatusConflict,
		"invalid_transition",
	},
	{store.ErrDuplicate, http.StatusConflict, "duplicate"},
	{store.ErrConflict, http.StatusConflict, "conflict"},
	{webhook.ErrInvalido, http.StatusBadRequest, "invalid"},
	{errEntrada, http.StatusBadRequest, "invalid"},
	{context.DeadlineExceeded, http.StatusGatewayTimeout, "timeout"},
	// O cliente desistiu: ninguém vai ler a resposta, e não é defeito
	// do servidor. 499 é a convenção do nginx para o caso.
	{context.Canceled, 499, "canceled"},
}

var errEntrada = errors.New("entrada inválida")

func (a *API) erro(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range erroHTTP {
		if errors.Is(err, e.erro) {
			escrever(w, e.status, Erro{e.codigo, err.Error()})
			return
		}
	}
	a.Log.ErrorContext(r.Context(), "erro interno",
		slog.String("rota", r.Pattern), slog.Any("erro", err))
	escrever(w, http.StatusInternalServerError,
		Erro{"internal", "erro interno"})
}

// escrever responde com JSON.
func escrever(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, v)
}

// ler decodifica o corpo, recusando o que o json/v2 recusa: UTF-8
// inválido e nomes de campo repetidos.
func ler(r *http.Request, v any) error {
	if err := json.UnmarshalRead(r.Body, v); err != nil {
		return fmt.Errorf("%w: %w", errEntrada, err)
	}
	return nil
}

func namespace(ctx context.Context) string {
	ns, _ := ctx.Value(chaveNamespace{}).(string)
	return ns
}

type chaveNamespace struct{}

var agora = time.Now
