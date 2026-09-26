package main

import (
	"net/url"
	"strings"
)

// livro:inicio ctl-endpoint

// endpointAdd inscreve um endpoint de webhook. A CLI não sabe nada de
// banco nem de domínio: monta o corpo, chama a API e mostra a
// resposta — a mesma coisa que qualquer outro cliente faria.
func endpointAdd(c *cliente, args []string) error {
	fs := novoFlagSet("webhook endpoint add")
	u := fs.String("url", "", "URL que recebe os webhooks")
	eventos := fs.String("events", "",
		"tipos de evento, separados por vírgula; vazio: todos")
	segredo := fs.String("secret-ref", "", "referência ao segredo, "+
		"como env:NOME")
	desc := fs.String("description", "", "descrição")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "url", "secret-ref"); err != nil {
		return err
	}
	tipos := []string{}
	for t := range strings.SplitSeq(*eventos, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tipos = append(tipos, t)
		}
	}
	return c.chamar("POST", "/v1/webhooks/endpoints", map[string]any{
		"url": *u, "event_types": tipos, "secret_ref": *segredo,
		"description": *desc})
}

// livro:fim ctl-endpoint

func endpointList(c *cliente, args []string) error {
	fs := novoFlagSet("webhook endpoint list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return c.chamar("GET", "/v1/webhooks/endpoints", nil)
}

func endpointRemove(c *cliente, args []string) error {
	fs := novoFlagSet("webhook endpoint remove")
	eid := fs.String("id", "", "id do endpoint")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "id"); err != nil {
		return err
	}
	return c.chamar("DELETE",
		"/v1/webhooks/endpoints/"+url.PathEscape(*eid), nil)
}
