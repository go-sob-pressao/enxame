-- Agendamentos periódicos (cron).
--
-- Cada disparo vira um job com unique_key = 'cron:' || schedule_id || ':' ||
-- instante previsto. Se o nó que disparou morrer antes de avançar
-- next_fire_at, o próximo dono dispara de novo e o índice job_unico absorve
-- a duplicata: disparo exatamente uma vez por janela, sem depender de relógio.
CREATE TABLE schedule (
    namespace     TEXT        NOT NULL,
    schedule_id   TEXT        NOT NULL,
    cron_expr     TEXT        NOT NULL,
    timezone      TEXT        NOT NULL DEFAULT 'UTC',
    queue         TEXT        NOT NULL,
    kind          TEXT        NOT NULL,
    args          JSONB       NOT NULL DEFAULT '{}',
    overlap       TEXT        NOT NULL DEFAULT 'skip'
                  CHECK (overlap IN ('skip', 'allow', 'buffer_one')),
    paused        BOOLEAN     NOT NULL DEFAULT false,
    next_fire_at  TIMESTAMPTZ NOT NULL,
    last_fired_at TIMESTAMPTZ,
    PRIMARY KEY (namespace, schedule_id)
);

CREATE INDEX schedule_proximo ON schedule (next_fire_at) WHERE NOT paused;
