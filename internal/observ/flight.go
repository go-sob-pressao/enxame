package observ

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"
	"time"
)

// livro:inicio voo

// Voo mantém o flight recorder do runtime ligado — os últimos segundos
// do trace de execução, sempre, em memória — e grava essa janela num
// arquivo quando uma requisição passa do limiar. O trace de uma
// latência rara fica disponível depois que ela aconteceu, sem ninguém
// ter ligado nada antes.
type Voo struct {
	Limiar    time.Duration // a requisição que passa disto dispara
	Dir       string        // onde os traces são gravados
	Intervalo time.Duration // no máximo uma gravação por intervalo
	Log       *slog.Logger

	fr     *trace.FlightRecorder
	mu     sync.Mutex
	ultima time.Time
}

// Ligar começa a gravar a janela: 10 s ou 32 MiB, o que vier antes.
func (v *Voo) Ligar() error {
	v.fr = trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge: 10 * time.Second, MaxBytes: 32 << 20})
	return v.fr.Start()
}

// Desligar para o flight recorder.
func (v *Voo) Desligar() { v.fr.Stop() }

// Middleware mede cada requisição e grava a janela se ela passar do
// limiar.
func (v *Voo) Middleware(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		inicio := time.Now()
		prox.ServeHTTP(w, r)
		if d := time.Since(inicio); d > v.Limiar {
			v.gravar(r, d)
		}
	})
}

// gravar escreve a janela, se a última gravação já tem Intervalo. A
// escrita acontece depois da resposta: quem esperou não espera mais.
func (v *Voo) gravar(r *http.Request, d time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Since(v.ultima) < v.Intervalo {
		return
	}
	v.ultima = time.Now()
	arq := filepath.Join(v.Dir, fmt.Sprintf("voo-%s.trace",
		v.ultima.Format("20060102-150405.000")))
	f, err := os.Create(arq) //nolint:gosec // o diretório é do operador
	if err == nil {
		_, err = v.fr.WriteTo(f)
		err = errors.Join(err, f.Close())
	}
	v.Log.WarnContext(r.Context(), "requisição lenta: trace gravado",
		slog.String("rota", r.Pattern), slog.Duration("duracao", d),
		slog.String("arquivo", arq), slog.Any("erro", err))
}

// livro:fim voo
