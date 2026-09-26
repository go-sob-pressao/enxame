# Capítulo 19 — API pública HTTP

| Diretório | O que mostra |
|---|---|
| `json/` | as armadilhas do JSON — ausente × zero, `omitempty`, caixa das chaves, chave repetida, UTF-8 inválido — no `encoding/json` e no `encoding/json/v2`; o Custo Real #6 |
| `desligamento/` | o enigma: o `Shutdown` que espera um long-poll até o prazo (`-tags defeito`). A versão correta é a da API do Enxame, `internal/transport/http/servidor.go` |

```bash
go test -v ./examples/cap19/json/
go test -run '^$' -bench . -benchmem ./examples/cap19/json/
go test -tags defeito -v ./examples/cap19/desligamento/
```

A API está em `internal/transport/http`; o servidor, em `cmd/enxamed`; a
CLI, em `cmd/enxamectl`; a descrição OpenAPI, gerada, em `api/openapi.json`.
