package main

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

// visao é um cluster fixo: a visão de um nó que vê um vivo e um
// suspeito.
type visao struct{}

func (visao) No() coordinator.NodeID { return "no-1" }

func (visao) Visao() []membership.Situacao {
	return []membership.Situacao{
		{No: "no-2", Estado: membership.Vivo, Phi: 0.2,
			Silencio: 300 * time.Millisecond},
		{No: "no-3", Estado: membership.Suspeito, Phi: 4.1,
			Silencio: 4 * time.Second},
	}
}

func TestClusterMembers(t *testing.T) {
	a := api.NovaAPI(nil, map[string]string{"tk": "loja"},
		slog.New(slog.DiscardHandler))
	a.Cluster = visao{}
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()
	env := func(k string) string {
		return map[string]string{"ENXAME_API": srv.URL,
			"ENXAME_TOKEN": "tk"}[k]
	}
	var saida, erros bytes.Buffer
	if code := executar([]string{"cluster", "members"}, &saida, &erros,
		env); code != 0 {
		t.Fatalf("código %d: %s", code, erros.String())
	}
	t.Log("\n" + saida.String())
	for _, linha := range []string{"no-1  vivo", "no-2  vivo",
		"no-3  suspeito  4.1"} {
		if !strings.Contains(saida.String(), linha) {
			t.Fatalf("falta %q", linha)
		}
	}
}
