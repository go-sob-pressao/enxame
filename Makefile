# Enxame — alvos de desenvolvimento. `make help` lista todos.
#
# Os alvos sim e integration informam SKIP explicitamente enquanto não existem
# cenários: um alvo que "passa" sem testar nada é um verde mentiroso.

GO      ?= go
VERSAO  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.versao=$(VERSAO)
PKGS    := ./...

.DEFAULT_GOAL := help
.PHONY: help check build test race lint fmt arch vuln sim chaos integration fuzz cover fix tidy up down proto clean defeitos

help:            ## lista os alvos
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-12s %s\n",$$1,$$2}'

check: lint arch race  ## o que a CI exige antes de qualquer tag de capítulo

build:           ## compila os binários em bin/
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/enxamed   ./cmd/enxamed
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/enxamectl ./cmd/enxamectl

test:            ## testes unitários
	$(GO) test $(PKGS)

race:            ## testes com race detector (obrigatório na CI — Pacto 3)
	$(GO) test -race -count=1 $(PKGS)

lint:            ## golangci-lint v2 (.golangci.yml)
	golangci-lint run

fmt:             ## formata o código
	golangci-lint fmt

arch:            ## regras de dependência entre camadas (tools/archcheck)
	$(GO) run ./tools/archcheck

vuln:            ## vulnerabilidades conhecidas nas dependências e na stdlib
	$(GO) tool govulncheck $(PKGS)

sim:             ## cenários de simulação determinística (Cap. 27)
	@if ls test/simulation/*_test.go >/dev/null 2>&1; then \
		$(GO) test -tags=simulation -timeout=30m ./test/simulation/...; \
	else echo "sim: SKIP — nenhum cenário ainda; entram no Capítulo 27"; fi

chaos:           ## experimentos de caos contra cluster real (Cap. 28)
	@if ls test/chaos/*_test.go >/dev/null 2>&1; then \
		$(GO) test -tags=chaos -timeout=30m ./test/chaos/...; \
	else echo "chaos: SKIP — nenhum experimento ainda; entram no Capítulo 28"; fi

integration:     ## integração contra Postgres real (make up; ENXAME_DB_DSN)
	@if ls test/integration/*_test.go >/dev/null 2>&1; then \
		$(GO) test -tags=integration -count=1 ./test/integration/...; \
	else echo "integration: SKIP — nenhum teste ainda; entram no Capítulo 12"; fi

fuzz:            ## fuzzing de 30s por alvo (Cap. 13)
	@$(GO) test -list '^Fuzz' $(PKGS) | grep -E '^Fuzz' || echo "fuzz: SKIP — nenhum alvo ainda; entram no Capítulo 13"

defeitos:        ## reproduz os defeitos dos exemplos (build tag defeito; fora da CI)
	$(GO) test -count=1 -short -tags defeito ./examples/...

cover:           ## relatório de cobertura em HTML
	$(GO) test -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -html=coverage.out

fix:             ## aplica os modernizers do go fix (Go 1.26+)
	$(GO) fix $(PKGS)

tidy:
	$(GO) mod tidy

up:              ## sobe Postgres, Jaeger, Prometheus e Grafana locais
	docker compose -f deploy/docker/compose.yaml up -d --wait

down:
	docker compose -f deploy/docker/compose.yaml down -v

proto:           ## gera código gRPC com buf (Cap. 18)
	@if find internal/transport/grpc/proto -name '*.proto' | grep -q .; then buf generate; \
	else echo "proto: SKIP — nenhum .proto ainda; entram no Capítulo 18"; fi

clean:
	rm -rf bin dist coverage.out
