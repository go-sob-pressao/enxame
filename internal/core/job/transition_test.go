package job_test

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

var t0 = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

var idFixo = id.JobID(
	uuid.MustParse("0192a3b4-0000-7000-8000-000000000001"),
)

func novo(t *testing.T, s job.Spec) job.Job {
	t.Helper()
	if s.ID.IsZero() {
		s.ID = idFixo
	}
	if s.Queue == "" {
		s.Queue = "padrao"
	}
	if s.Kind == "" {
		s.Kind = "eco"
	}
	evs, j, err := job.Insert(s, t0)
	if err != nil {
		t.Fatal(err)
	}
	j, err = job.ApplyAll(j, evs)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func aplicar(
	t *testing.T,
	j job.Job,
	evs []job.Event,
	err error,
) job.Job {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	j, err = job.ApplyAll(j, evs)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestCaminhoFeliz(t *testing.T) {
	j := novo(t, job.Spec{})
	if j.State != job.StateAvailable {
		t.Fatalf("estado %q", j.State)
	}
	evs, err := job.Start(j, t0, "w1")
	j = aplicar(t, j, evs, err)
	evs, err = job.Complete(j, t0.Add(time.Second))
	j = aplicar(t, j, evs, err)
	if j.State != job.StateCompleted || j.Attempt != 1 ||
		!j.FinalizedAt.Equal(t0.Add(time.Second)) {
		t.Fatalf("job %+v", j)
	}
}

func TestFalhaComRetryDepoisDescarte(t *testing.T) {
	j := novo(t, job.Spec{MaxAttempts: 2})
	for tentativa := 1; tentativa <= 2; tentativa++ {
		if tentativa > 1 {
			evs, err := job.MakeAvailable(j, j.ScheduledAt)
			j = aplicar(t, j, evs, err)
		}
		evs, err := job.Start(j, t0, "w1")
		j = aplicar(t, j, evs, err)
		evs, err = job.Fail(
			j,
			t0,
			"gateway fora do ar",
			false,
			t0.Add(time.Minute),
		)
		j = aplicar(t, j, evs, err)
	}
	if j.State != job.StateDiscarded ||
		j.LastError != "gateway fora do ar" {
		t.Fatalf("depois de 2 falhas: %+v", j)
	}
}

func TestAgendadoParaOFuturo(t *testing.T) {
	j := novo(t, job.Spec{RunAt: t0.Add(time.Hour)})
	if j.State != job.StateScheduled {
		t.Fatalf("estado %q", j.State)
	}
	if _, err := job.MakeAvailable(j, t0); !errors.Is(
		err,
		job.ErrInvalidTransition,
	) {
		t.Fatalf("antes da hora deveria falhar, veio %v", err)
	}
}

func TestTransicoesInvalidas(t *testing.T) {
	j := novo(t, job.Spec{})
	if _, err := job.Complete(j, t0); !errors.Is(
		err,
		job.ErrInvalidTransition,
	) {
		t.Errorf("concluir sem iniciar: %v", err)
	}
	evs, err := job.Start(j, t0, "w1")
	j = aplicar(t, j, evs, err)
	if _, err := job.Cancel(j, t0); !errors.Is(
		err,
		job.ErrInvalidTransition,
	) {
		t.Errorf("cancelar em execução: %v", err)
	}
}

// Achado pela propriedade do Capítulo 13: resgatar a última tentativa
// devolvia o job à fila, e a tentativa seguinte passava do máximo.
func TestResgateNaUltimaTentativaDescarta(t *testing.T) {
	j := novo(t, job.Spec{MaxAttempts: 1})
	evs, err := job.Start(j, t0, "w1")
	j = aplicar(t, j, evs, err)
	evs, err = job.Rescue(j, t0.Add(time.Minute))
	j = aplicar(t, j, evs, err)
	if j.State != job.StateDiscarded || j.Attempt != 1 {
		t.Fatalf("depois do resgate da última tentativa: %+v", j)
	}
}
