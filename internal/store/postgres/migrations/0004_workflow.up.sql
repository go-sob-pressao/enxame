-- Workflows por passos memoizados (ADR-002).
--
-- Um run é uma sequência de passos. No replay, cada passo já concluído
-- devolve o output gravado aqui em vez de executar de novo. Cada passo do
-- tipo 'call' é executado por um job; 'sleep' é um job agendado para wake_at.
CREATE TABLE workflow_run (
    run_id        UUID        PRIMARY KEY DEFAULT uuidv7(),
    partition_id  INT         NOT NULL CHECK (partition_id >= 0),
    namespace     TEXT        NOT NULL,
    workflow_id   TEXT        NOT NULL,   -- id de negócio, escolhido pela aplicação
    workflow_type TEXT        NOT NULL,
    code_version  INT         NOT NULL,   -- versão do código que iniciou o run (Cap. 17)
    queue         TEXT        NOT NULL,
    state         TEXT        NOT NULL DEFAULT 'running'
                  CHECK (state IN ('running', 'completed', 'failed', 'cancelled')),
    input         JSONB       NOT NULL DEFAULT '{}',
    output        JSONB,
    error         JSONB,
    next_step_seq INT         NOT NULL DEFAULT 1,
    version       BIGINT      NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at     TIMESTAMPTZ,

    CONSTRAINT workflow_run_fechado_coerente CHECK ((state = 'running') = (closed_at IS NULL))
);

-- Start idempotente: no máximo um run aberto por workflow_id.
-- Sem este índice, dois StartWorkflow concorrentes criariam dois runs.
CREATE UNIQUE INDEX workflow_run_aberto ON workflow_run (namespace, workflow_id)
    WHERE state = 'running';

CREATE INDEX workflow_run_consulta ON workflow_run (namespace, state, started_at DESC);

CREATE TABLE workflow_step (
    run_id       UUID        NOT NULL REFERENCES workflow_run (run_id) ON DELETE CASCADE,
    step_seq     INT         NOT NULL CHECK (step_seq > 0),
    step_name    TEXT        NOT NULL,   -- comparado no replay: divergência = NonDeterministicError
    step_kind    TEXT        NOT NULL CHECK (step_kind IN ('call', 'sleep', 'signal', 'side_effect')),
    state        TEXT        NOT NULL CHECK (state IN ('pending', 'completed', 'failed')),
    output       JSONB,
    error        JSONB,
    job_id       UUID,       -- job que executa o passo
    wake_at      TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (run_id, step_seq)
);

CREATE TABLE workflow_signal (
    run_id           UUID        NOT NULL REFERENCES workflow_run (run_id) ON DELETE CASCADE,
    signal_seq       BIGINT      GENERATED ALWAYS AS IDENTITY,
    name             TEXT        NOT NULL,
    payload          JSONB       NOT NULL DEFAULT '{}',
    received_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    consumed_by_step INT,
    PRIMARY KEY (run_id, signal_seq)
);
