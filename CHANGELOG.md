# Mudanças

O formato segue o [Keep a Changelog](https://keepachangelog.com/pt-BR/),
e as versões, o [versionamento semântico](https://semver.org/lang/pt-BR/).

A partir da `v1.0.0`, a compatibilidade vale para o que está em `pkg/`
(a biblioteca), para a API HTTP de `api/openapi.json`, para o protocolo
gRPC dos workers (`buf breaking` na CI) e para o esquema do banco, que só
muda por migrações do tipo expandir e depois contrair. O que está em
`internal/` pode mudar em qualquer versão.

## [1.0.0] — 2026-09-28

A primeira versão estável: o Enxame do fim do livro, marco a marco.

### Adicionado

- **M0** (Cap. 2): o domínio puro — a máquina de estados do job, a fila
  em memória e o executor de um job por vez —, com as regras de camada
  no `archcheck`.
- **M1** (Cap. 6): o pool de workers com limite, `recover` por
  tentativa e prazo por tentativa.
- **M2** (Cap. 9): o pool sem corridas nem vazamentos, conferido pelo
  `-race` e pelo `goleak`.
- **M3** (Cap. 17): o estado no PostgreSQL e no SQLite, enfileiramento
  transacional, idempotência, workflows duráveis com versionamento e
  migrações sem parada.
- **M4** (Cap. 21): gRPC para workers remotos, a API HTTP pública,
  retry com backoff e jitter, circuit breaker, descarte de carga.
- **M5** (Cap. 26): o cluster — membership, liderança, 512 partições
  com fencing e rebalanceamento.
- **M6** (Cap. 29): simulação determinística, que achou o
  rebalanceamento perdendo jobs e o corrigiu; experimentos de caos;
  caminho quente medido e flight recorder.
- **M7** (Cap. 32): traces, métricas e logs correlacionados; imagem
  reproduzível com SBOM e proveniência; manifestos do Kubernetes e chart
  do Helm; SLOs com alarmes pelo ritmo de queima; runbooks.
- `WorkerConfig.AttemptTimeout`: o prazo de cada tentativa, separado do
  prazo de resgate.

### Mudado

- `WorkerConfig.RescueAfter` deixou de ser também o prazo de cada
  tentativa, e o padrão passou a ser `AttemptTimeout` + 1 min. `Run`
  devolve `ErrConfig` quando `RescueAfter` não é maior que
  `AttemptTimeout`: uma tentativa ainda no prazo seria resgatada e
  rodaria de novo (a Missão #7 do livro).

### Corrigido

- Um membership que ainda não leu a tabela de membros devolve
  `ErrSemVisao` em vez de "só eu", e um líder recém-reiniciado não grava
  mais um mapa com todas as partições para si (Cap. 32).

### Fora desta versão

- Sinais externos para workflows: a tabela `workflow_signal` existe
  desde a migração 0004, reservada; a API (`WaitSignal`) não. O
  `README` e a documentação do pacote deixaram de anunciá-la.

As versões anteriores não foram publicadas como versões: são as tags
`cap-NN`, uma por capítulo do livro.
