# Checklist de Produção

Quarenta itens, quatro por degrau da escada *Compila ≠ Está correto*
(Capítulo 31). Cada item é verificável: um comando, um teste ou um
arquivo que diz sim ou não. A coluna "Enxame" é o estado na tag
`cap-32`.

| # | Degrau | Item | Como se verifica | Enxame |
|---|---|---|---|---|
| 1 | Compila | Lint com zero avisos, com a configuração no repositório | `make lint` | sim |
| 2 | Compila | `go.mod` e `go.sum` em dia | `go mod tidy -diff` vazio | sim |
| 3 | Compila | Uma versão de Go só: `toolchain` no `go.mod`, a mesma na imagem de build e na CI | `go.mod`, `Dockerfile`, `go-version-file` | sim |
| 4 | Compila | Nada para o `go fix` modernizar | `go fix -diff ./...` vazio | sim |
| 5 | Passa nos testes | Testes de integração contra o banco de verdade, na CI | `make integration` | sim |
| 6 | Passa nos testes | Uma suíte de contrato para toda interface com mais de uma implementação | `storetest`; `Conformidade` | sim |
| 7 | Passa nos testes | Nenhum `time.Sleep` em teste; relógio injetável ou `synctest` | lint (`forbidigo`) | sim |
| 8 | Passa nos testes | Fuzzing dos parsers, com o corpus guardado | `make fuzz`; job noturno | sim |
| 9 | Livre de corridas | Todo teste roda com `-race` na CI | `make race` | sim |
| 10 | Livre de corridas | Ordem dos testes embaralhada, repetida | `-shuffle=on`, 10 execuções (Painel) | sim |
| 11 | Livre de corridas | Nenhuma trava global no caminho quente; travas por chave | benchmark do striping (Cap. 8) | sim |
| 12 | Livre de corridas | As camadas não importam o que não devem | `make arch` | sim |
| 13 | Sem vazamento | `goleak` no `TestMain` dos pacotes com goroutines | `main_test.go` | sim |
| 14 | Sem vazamento | Toda goroutine termina com o contexto de quem a criou | revisão + `goleak` | sim |
| 15 | Sem vazamento | Perfis `goroutine` e `goroutineleak` acessíveis em produção, fora da API | `-diag` | sim |
| 16 | Sem vazamento | Desligamento gracioso testado: `SIGTERM` drena e termina no prazo | teste do `enxamed` (Cap. 19) | sim |
| 17 | Durável | Enfileirar na mesma transação dos dados de negócio | `InsertTx`, `PublishTx` | sim |
| 18 | Durável | Fencing: só a tentativa vigente conclui; só o dono da partição escreve | `ErrCercado`, testes do zumbi | sim |
| 19 | Durável | Migrações versionadas, com ida e volta testadas | `migrations/`, suíte de integração | sim |
| 20 | Durável | Efeitos idempotentes pela chave do job | `IdempotencyKey`, Cap. 16 | sim |
| 21 | Resiliente | Prazo em toda chamada que sai do processo | `Prazo`, `AttemptTimeout`, contexto | sim |
| 22 | Resiliente | Retry com backoff e jitter; erro permanente classificado como tal | `policy.Retry`, `Permanent` | sim |
| 23 | Resiliente | Descarte de carga e fila limitada, com `503`/`429` e `Retry-After` | `-max-em-curso`, `-max-na-fila` | sim |
| 24 | Resiliente | Uma falha passageira de dependência não derruba o processo | `Supervisionar`, Experimento 1 (Cap. 28) | sim |
| 25 | Distribuído | Decisões de tempo pelo relógio do banco, nunca pelo do nó | `RelogioDoBanco`, `TestDisparoComRelogioTorto` | sim |
| 26 | Distribuído | Um dono por partição, conferido pelo banco | `range_id` + `FOR SHARE` | sim |
| 27 | Distribuído | Rebalanceamento sem perda, sob falhas sorteadas | `make sim` | sim |
| 28 | Distribuído | Ordem por chave garantida por uma regra que o banco confere | condição de cabeça + índice único | sim |
| 29 | Comprovado | Simulação determinística com seeds fixas na CI e varredura noturna | `ci.yaml`, `nightly-sim.yaml` | sim |
| 30 | Comprovado | Experimentos de caos com hipótese refutável | `make chaos` | sim |
| 31 | Comprovado | Benchmarks do caminho quente comparados com `benchstat` | `BenchmarkInserirJob` | sim |
| 32 | Comprovado | Build reproduzível e nenhuma vulnerabilidade conhecida alcançável | `make reproduzivel`, `make vuln` | sim |
| 33 | Observável | Traces com o contexto gravado no job | `trace_parent`, `TestTraceAtravessaOBanco` | sim |
| 34 | Observável | Métricas RED, USE e de domínio, sem rótulo de cardinalidade ilimitada | `/metrics`, revisão dos rótulos | sim |
| 35 | Observável | Logs estruturados com `trace_id` e `job_id` | `logging.Correlacao` | sim |
| 36 | Observável | Todo alarme diz o próximo passo, e tem teste | `alertas.yml`, `make alertas` | sim |
| 37 | Operável | Imagem mínima, sem root, com SBOM e proveniência | `Dockerfile`, `release.yaml` | sim |
| 38 | Operável | Probes que dizem a verdade: nenhuma depende de uma dependência compartilhada | `statefulset.yaml`, `TestStatusz` | sim |
| 39 | Operável | Memória com limite e `GOMEMLIMIT` abaixo dele; CPU pedida, e o `GOMAXPROCS` que o runtime leu no log | `statefulset.yaml`, log "no ar" | sim |
| 40 | Operável | SLO, runbook para cada alarme, dono declarado e modelo de postmortem | `docs/slo.md`, `docs/runbook/`, `docs/dono.md` | sim |
