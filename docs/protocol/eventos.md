# Catálogo de eventos

Histórico append-only. `seq` é monotônico por entidade, alocado de
`job.next_seq` (ou `workflow_run.next_step_seq`) dentro da mesma transação que
muda o estado. Sem lacunas, sem duplicatas.

## Job (`job_event`)

| Evento | Quando |
|---|---|
| `JobInserted` | enfileirado (`Insert`/`InsertTx`), com `unique_key` se houver |
| `JobScheduled` | agendado para o futuro, ou retry agendado com backoff |
| `JobMadeAvailable` | o timer pump o tornou elegível |
| `AttemptStarted` | um worker o pegou; registra `attempted_by` e a tentativa |
| `AttemptHeartbeat` | job longo sinalizou que está vivo (amostrado) |
| `AttemptFailed` | erro, pânico recuperado ou timeout, com a causa |
| `JobCompleted` | concluído |
| `JobDiscarded` | tentativas esgotadas ou erro marcado como permanente |
| `JobCancelled` | cancelado pela API ou pelo run de workflow |
| `JobRescued` | tentativa órfã (worker morto) devolvida à fila |

## Workflow (`workflow_step`, `workflow_signal`)

| Evento | Quando |
|---|---|
| `RunStarted` | `StartWorkflow`; no máximo um run aberto por `workflow_id` |
| `StepScheduled` / `StepCompleted` / `StepFailed` | ciclo de um passo `call` |
| `SleepStarted` / `SleepFired` | passo `sleep` (job agendado) |
| `SignalReceived` / `SignalConsumed` | entrada externa e o passo que a consumiu |
| `SideEffectRecorded` | valor não determinístico congelado |
| `RunCompleted` / `RunFailed` / `RunCancelled` | encerramento |
| `NonDeterminismDetected` | nome do passo N diverge do gravado; run parado para intervenção |

## Webhook (`webhook_attempt`)

| Evento | Quando |
|---|---|
| `MessagePublished` | `Publish`/`PublishTx`, com `idempotency_key` |
| `FanoutCompleted` | um job de entrega criado por endpoint inscrito |
| `DeliveryAttempted` | tentativa com código HTTP, duração e trecho da resposta |
| `EndpointDisabled` | falhas persistentes além da política; exige reativação |
