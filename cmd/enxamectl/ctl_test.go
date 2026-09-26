//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// A CLI contra a API de verdade.
func TestCLIContraAPI(t *testing.T) {
	a := api.NovaAPI(testutil.Postgres(t),
		map[string]string{"tk": "loja"}, slog.New(slog.DiscardHandler))
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()
	env := func(k string) string {
		return map[string]string{"ENXAME_API": srv.URL,
			"ENXAME_TOKEN": "tk"}[k]
	}
	rodar := func(args ...string) map[string]any {
		t.Helper()
		var saida, erros bytes.Buffer
		if code := executar(args, &saida, &erros, env); code != 0 {
			t.Fatalf("%v: código %d: %s", args, code, erros.String())
		}
		var m map[string]any
		_ = json.Unmarshal(saida.Bytes(), &m)
		return m
	}
	j := rodar("job", "insert", "--queue", "q", "--kind", "eco",
		"--args", `{"n":1}`)
	jid, _ := j["id"].(string)
	if j = rodar("job", "cancel", "--id", jid); j["state"] != "cancelled" {
		t.Fatalf("cancel: %v", j)
	}
	e := rodar("webhook", "endpoint", "add", "--url",
		"https://cliente.exemplo/hooks", "--events", "pedido.pago",
		"--secret-ref", "env:SEGREDO")
	eid, _ := e["id"].(string)
	l := rodar("webhook", "endpoint", "list")
	if !strings.Contains(stringDe(l), eid) {
		t.Fatalf("list: %v", l)
	}
	rodar("webhook", "endpoint", "remove", "--id", eid)
	var saida, erros bytes.Buffer
	if code := executar([]string{"webhook", "endpoint", "remove", "--id",
		eid}, &saida, &erros, env); code != 1 {
		t.Fatalf("remover de novo: %d", code)
	}
	t.Logf("segundo remove: %s", strings.TrimSpace(erros.String()))
}

func stringDe(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
