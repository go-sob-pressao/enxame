package queue_test

import (
	"testing"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
)

var t0 = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

func jid(n byte) id.JobID {
	u := uuid.MustParse("0192a3b4-0000-7000-8000-000000000000")
	u[15] = n
	return id.JobID(u)
}

func TestFetchRespeitaPrioridadeEOrdem(t *testing.T) {
	m := queue.NewMemory()
	for i, p := range []int{3, 1, 1} {
		s := job.Spec{
			ID:       jid(byte(i + 1)),
			Queue:    "q",
			Kind:     "eco",
			Priority: p,
		}
		if _, err := m.Insert(s, t0.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	var ordem []id.JobID
	for {
		j, ok, err := m.Fetch("q", t0.Add(time.Minute), "w")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		ordem = append(ordem, j.ID)
	}
	quer := []id.JobID{jid(2), jid(3), jid(1)}
	if len(ordem) != 3 || ordem[0] != quer[0] || ordem[1] != quer[1] ||
		ordem[2] != quer[2] {
		t.Fatalf("ordem %v, quer %v", ordem, quer)
	}
}

func TestRetryVoltaDepoisDoBackoff(t *testing.T) {
	m := queue.NewMemory()
	if _, err := m.Insert(job.Spec{ID: jid(1), Queue: "q", Kind: "eco"}, t0); err != nil {
		t.Fatal(err)
	}
	j, _, _ := m.Fetch("q", t0, "w")
	if err := m.Fail(j.ID, t0, "falhou", false, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n, _ := m.Promote(t0.Add(30 * time.Second)); n != 0 {
		t.Fatalf("promoveu %d antes da hora", n)
	}
	if n, _ := m.Promote(t0.Add(time.Minute)); n != 1 {
		t.Fatalf("promoveu %d na hora", n)
	}
	_, hist, _ := m.Get(j.ID)
	if len(hist) != 5 {
		t.Fatalf("histórico com %d eventos: %v", len(hist), hist)
	}
}
