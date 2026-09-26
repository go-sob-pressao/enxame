package main

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// cliente fala com a API.
type cliente struct {
	api, token string
	saida      io.Writer
}

// chamar faz a requisição e imprime a resposta formatada. Um status de
// erro vira erro, com a mensagem que a API mandou.
func (c *cliente) chamar(metodo, caminho string, corpo any) error {
	b, err := c.fazer(metodo, caminho, corpo)
	if err != nil || len(b) == 0 {
		return err
	}
	v := jsontext.Value(b)
	if err := v.Indent(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(c.saida, string(v))
	return err
}

// fazer faz a requisição e devolve o corpo da resposta.
func (c *cliente) fazer(metodo, caminho string, corpo any) ([]byte,
	error) {
	var r io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	ctx, cancel := context.WithTimeout(context.Background(),
		2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, metodo,
		c.api+caminho, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &e)
		return nil, fmt.Errorf("%s: %s", resp.Status, e.Message)
	}
	return b, nil
}

// novoFlagSet cria o FlagSet de um subcomando, que não imprime nada
// sozinho: o erro volta para executar.
func novoFlagSet(nome string) *flag.FlagSet {
	fs := flag.NewFlagSet(nome, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// jsonDe lê um valor JSON literal, ou de um arquivo com @caminho.
func jsonDe(s string) (jsontext.Value, error) {
	if s == "" {
		return nil, nil
	}
	b := []byte(s)
	if caminho, ok := strings.CutPrefix(s, "@"); ok {
		var err error
		if b, err = os.ReadFile(caminho); err != nil {
			return nil, err
		}
	}
	v := jsontext.Value(b)
	if !v.IsValid() {
		return nil, fmt.Errorf("JSON inválido: %.40s", s)
	}
	return v, nil
}

func obrigatorio(fs *flag.FlagSet, nomes ...string) error {
	for _, n := range nomes {
		if fs.Lookup(n).Value.String() == "" {
			return fmt.Errorf("--%s é obrigatório", n)
		}
	}
	return nil
}
