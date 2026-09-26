package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// livro:inicio openapi

// OpenAPI gera a descrição da API a partir da tabela de rotas e dos
// tipos Go de entrada e saída. Ninguém a escreve à mão, e ela não tem
// como ficar desatualizada: um teste compara o arquivo do repositório
// com o que este código gera, e falha na primeira rota nova que não o
// atualizou.
func (a *API) OpenAPI(versao string) ([]byte, error) {
	g := &gerador{esquemas: map[string]any{}}
	caminhos := map[string]map[string]any{}
	for _, r := range a.Rotas() {
		metodo, caminho, _ := strings.Cut(r.Padrao, " ")
		op := map[string]any{"summary": r.Resumo,
			"responses": map[string]any{}}
		if !r.Publica {
			op["security"] = []any{map[string]any{"token": []any{}}}
		}
		if r.Entrada != nil {
			op["requestBody"] = map[string]any{"required": true,
				"content": corpo(g.ref(reflect.TypeOf(r.Entrada)))}
		}
		resp := map[string]any{"description": http.StatusText(r.Status)}
		if r.Saida != nil {
			resp["content"] = corpo(g.ref(reflect.TypeOf(r.Saida)))
		}
		op["responses"].(map[string]any)[itoa(r.Status)] = resp
		if !r.Publica {
			g.recusas(op["responses"].(map[string]any))
		}
		if caminhos[caminho] == nil {
			caminhos[caminho] = map[string]any{}
		}
		caminhos[caminho][strings.ToLower(metodo)] = op
	}
	g.esquemas["Erro"] = g.esquema(reflect.TypeFor[Erro]())
	return json.Marshal(map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{"title": "Enxame API",
			"version": versao},
		"paths": caminhos,
		"components": map[string]any{"schemas": g.esquemas,
			"securitySchemes": map[string]any{"token": map[string]any{
				"type": "http", "scheme": "bearer"}}},
	}, json.Deterministic(true), jsontext.WithIndent("  "))
}

// livro:fim openapi

type gerador struct{ esquemas map[string]any }

// recusas documenta as respostas de toda rota autenticada que não vêm
// do handler: token inválido, taxa excedida e servidor sobrecarregado.
// As duas últimas dizem quando voltar.
func (g *gerador) recusas(respostas map[string]any) {
	for _, s := range []int{401, 429, 503} {
		r := map[string]any{"description": http.StatusText(s),
			"content": corpo(g.ref(reflect.TypeFor[Erro]()))}
		if s != 401 {
			r["headers"] = map[string]any{"Retry-After": map[string]any{
				"description": "segundos até tentar de novo",
				"schema":      map[string]any{"type": "integer"}}}
		}
		respostas[itoa(s)] = r
	}
}

var tipoTempo = reflect.TypeFor[time.Time]()
var tipoValor = reflect.TypeFor[jsontext.Value]()
var naoAlfa = regexp.MustCompile(`[^A-Za-z0-9]`)

func nome(t reflect.Type) string {
	n := t.Name()
	if i := strings.Index(n, "["); i >= 0 { // Lista[pacote.Endpoint]
		base, arg := n[:i], n[i+1:len(n)-1]
		n = base + arg[strings.LastIndex(arg, ".")+1:]
	}
	return naoAlfa.ReplaceAllString(n, "")
}

func (g *gerador) ref(t reflect.Type) map[string]any {
	n := nome(t)
	if _, ok := g.esquemas[n]; !ok {
		g.esquemas[n] = nil // reserva, para tipos recursivos
		g.esquemas[n] = g.esquema(t)
	}
	return map[string]any{"$ref": "#/components/schemas/" + n}
}

func (g *gerador) esquema(t reflect.Type) map[string]any {
	switch t {
	case tipoTempo:
		return map[string]any{"type": "string", "format": "date-time"}
	case tipoValor:
		return map[string]any{} // qualquer JSON
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Slice:
		return map[string]any{"type": "array",
			"items": g.esquema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object",
			"additionalProperties": g.esquema(t.Elem())}
	case reflect.Struct:
		props, obrig := map[string]any{}, []string{}
		for f := range t.Fields() {
			tag, opcoes, _ := strings.Cut(f.Tag.Get("json"), ",")
			if !f.IsExported() || tag == "-" {
				continue
			}
			if tag == "" {
				tag = f.Name
			}
			if f.Type.Kind() == reflect.Struct && f.Type != tipoTempo {
				props[tag] = g.ref(f.Type)
			} else {
				props[tag] = g.esquema(f.Type)
			}
			if !strings.Contains(opcoes, "omitzero") {
				obrig = append(obrig, tag)
			}
		}
		return map[string]any{"type": "object", "properties": props,
			"required": obrig}
	}
	return map[string]any{}
}

func corpo(esquema map[string]any) map[string]any {
	return map[string]any{"application/json": map[string]any{
		"schema": esquema}}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
