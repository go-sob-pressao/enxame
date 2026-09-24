-- Histórico append-only de cada job: enfileirado, tentativa iniciada,
-- tentativa falhou (com o erro), concluído, descartado, cancelado.
-- É a fonte da verdade (ADR-001); job é a projeção.
--
-- seq é alocado de job.next_seq na mesma transação: sem lacunas, sem duplicatas.
CREATE TABLE job_event (
    job_id      UUID        NOT NULL REFERENCES job (job_id) ON DELETE CASCADE,
    seq         INT         NOT NULL CHECK (seq > 0),
    event_type  SMALLINT    NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload     JSONB       NOT NULL DEFAULT '{}',
    PRIMARY KEY (job_id, seq)
);
