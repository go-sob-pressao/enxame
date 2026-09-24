# ADR 0002 — Workflow por passos memoizados

- **Status:** aceita
- **Capítulo:** 2 (registro) · 14 e 17 (implementação)
- **Data:** 2026-09-24

## Contexto

O Enxame precisa de processos de várias etapas que sobrevivam a queda de
processo: cobrar, esperar confirmação por dois dias, emitir nota, notificar.
O leitor precisa aprender, com eles, determinismo, event sourcing,
idempotência e versionamento de código contra histórico antigo.

O modelo mais conhecido é o do Temporal: o worker reexecuta a função
inteira sobre o histórico, e `workflow.Go`/`workflow.NewChannel` rodam
num escalonador determinístico de corrotinas. Adotá-lo aqui teria dois
problemas:

1. **Custo desproporcional.** O escalonador de corrotinas é um subprojeto do
   tamanho de um capítulo, e o conhecimento que ele gera só se transfere para
   quem usa Temporal.
2. **Distância do leitor.** O modelo que o leitor encontra hoje no mercado
   fora do Temporal — DBOS, Inngest, Restate, Hatchet, funções duráveis de
   nuvem — é o de passos nomeados e memoizados.

## Opções consideradas

1. **Replay com escalonador de corrotinas (Temporal).** Máxima expressividade
   (concorrência dentro do workflow), máximo custo.
2. **Máquina de estados declarativa (Step Functions).** Simples de
   implementar, mas o workflow deixa de ser código Go — perde o ensino de
   determinismo.
3. **Passos memoizados.** O workflow é uma função Go comum. Cada
   `workflow.Step(ctx, "nome", fn)` grava o resultado em `workflow_step` na
   primeira execução; no replay, devolve o resultado gravado sem executar.

## Decisão

Opção 3.

- A função do workflow é reexecutada desde o início a cada avanço (replay),
  como no modelo do Temporal — determinismo continua obrigatório.
- Cada passo é identificado por `(run_id, step_seq)` e carrega `step_name`.
  No replay, se o passo de número N tiver nome diferente do gravado, o run
  para com `NonDeterministicError` — nunca é corrompido em silêncio.
- `Step` do tipo *call* é executado por um **job** (herda retry, timeout,
  idempotência e observabilidade da primitiva). *Sleep* é um job agendado.
  *WaitSignal* consome de `workflow_signal`. *SideEffect* grava o valor.
- Concorrência dentro do workflow fica restrita a `workflow.All(ctx, passos...)`
  — passos independentes em paralelo, resultado gravado em ordem de
  declaração. Não há goroutines nem channels de workflow.

## Consequências

- **Melhora:** dispensa o escalonador de corrotinas; aproxima o produto do que
  o leitor usa; toda execução de passo reaproveita o motor de jobs.
- **Mantém:** replay, determinismo, `NonDeterministicError`, versionamento
  (Cap. 17), event sourcing (Cap. 14).
- **Piora:** perde-se concorrência arbitrária dentro do workflow. Aceito: o
  caso comum é sequencial ou fan-out simples, coberto por `workflow.All`.
