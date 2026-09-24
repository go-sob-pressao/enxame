-- O job é a primitiva do sistema. Execuções de passo de workflow, disparos
-- de cron, sleeps e entregas de webhook são todos jobs — a plataforma usa a
-- própria primitiva.
--
-- Esta linha é a projeção do estado corrente; a fonte da verdade é job_event
-- (0003). Ambos mudam na mesma transação.
CREATE TABLE job (
    job_id       UUID        PRIMARY KEY DEFAULT uuidv7(),
    partition_id INT         NOT NULL CHECK (partition_id >= 0),
    namespace    TEXT        NOT NULL,
    queue        TEXT        NOT NULL,
    kind         TEXT        NOT NULL,
    args         JSONB       NOT NULL DEFAULT '{}',
    state        TEXT        NOT NULL DEFAULT 'available'
                 CHECK (state IN ('scheduled', 'available', 'running', 'retryable',
                                  'completed', 'discarded', 'cancelled')),
    priority     SMALLINT    NOT NULL DEFAULT 2 CHECK (priority BETWEEN 1 AND 4),
    attempt      INT         NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    max_attempts INT         NOT NULL DEFAULT 25 CHECK (max_attempts > 0),
    unique_key   TEXT,       -- idempotência de enfileiramento (Cap. 16)
    ordering_key TEXT,       -- FIFO por chave, ex.: o endpoint de um webhook (Cap. 26)
    run_id       UUID,       -- preenchido quando o job executa um passo de workflow
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    attempted_at TIMESTAMPTZ,
    attempted_by TEXT,
    finalized_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    next_seq     INT         NOT NULL DEFAULT 1,   -- próximo job_event.seq
    version      BIGINT      NOT NULL DEFAULT 0,   -- lock otimista (Cap. 15)

    CONSTRAINT job_finalizado_coerente CHECK (
        (state IN ('completed', 'discarded', 'cancelled')) = (finalized_at IS NOT NULL)
    ),
    CONSTRAINT job_tentativas_no_limite CHECK (attempt <= max_attempts)
);

-- Caminho quente da busca: SELECT … FOR UPDATE SKIP LOCKED sobre este índice.
CREATE INDEX job_busca ON job (partition_id, queue, priority, scheduled_at, job_id)
    WHERE state = 'available';

-- Timer pump: o que precisa virar 'available' quando o relógio chegar.
CREATE INDEX job_agendado ON job (partition_id, scheduled_at)
    WHERE state IN ('scheduled', 'retryable');

-- Resgate de jobs órfãos (worker morreu no meio da tentativa).
CREATE INDEX job_em_execucao ON job (partition_id, attempted_at)
    WHERE state = 'running';

-- Enfileirar duas vezes com a mesma unique_key não cria dois jobs.
-- Jobs descartados ou cancelados liberam a chave; concluídos, não.
CREATE UNIQUE INDEX job_unico ON job (namespace, unique_key)
    WHERE unique_key IS NOT NULL AND state NOT IN ('discarded', 'cancelled');

-- No máximo um job em execução por ordering_key: o banco impõe a ordem,
-- não a boa vontade do worker.
CREATE UNIQUE INDEX job_ordem_por_chave ON job (namespace, ordering_key)
    WHERE ordering_key IS NOT NULL AND state = 'running';

CREATE INDEX job_consulta ON job (namespace, state, created_at DESC);
