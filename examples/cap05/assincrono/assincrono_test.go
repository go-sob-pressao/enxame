package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// resultado faz uma requisição real (o servidor cancela o contexto
// quando o handler retorna, como em produção) e devolve o erro do
// processamento.
func resultado(t *testing.T) error {
	t.Helper()
	feito := make(chan error, 1)
	mux := http.NewServeMux()
	mux.Handle("POST /pedidos/{id}", aceitar(feito))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		srv.URL+"/pedidos/42",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", resp.StatusCode)
	}
	return <-feito
}
