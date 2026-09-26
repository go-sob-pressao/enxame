// Package custo é o Custo Real #5: a mesma chamada — o batimento de um
// job — por gRPC e por HTTP com JSON, na mesma máquina.
package custo

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// Motor responde a tudo na hora: o benchmark mede o transporte.
type Motor struct{}

// Heartbeat aceita o batimento.
func (Motor) Heartbeat(
	context.Context, id.JobID, time.Time, int,
) error {
	return nil
}

// Claim nunca tem job.
func (Motor) Claim(context.Context, string, time.Time,
	string) (job.Job, bool, error) {
	return job.Job{}, false, nil
}

// Decide não decide nada.
func (Motor) Decide(context.Context, id.JobID,
	func(job.Job) ([]job.Event, error)) (job.Job, error) {
	return job.Job{}, nil
}

// Batimento é o corpo da versão HTTP.
type Batimento struct {
	JobID   string `json:"job_id"`
	Attempt int    `json:"attempt"`
}

// HandlerHTTP é a mesma chamada, como API HTTP com JSON.
func HandlerHTTP(m Motor) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/heartbeat",
		func(w http.ResponseWriter, r *http.Request) {
			var b Batimento
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			jid, err := id.ParseJobID(b.JobID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := m.Heartbeat(r.Context(), jid, time.Now(),
				b.Attempt); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{}"))
		})
	return mux
}
