-- O mesmo modelo do Postgres (migrations 0002 e 0003), nos tipos do
-- SQLite: UUID como texto, instantes em nanossegundos Unix, JSON como
-- texto. O índice único parcial é idêntico.
CREATE TABLE IF NOT EXISTS job (
    job_id       TEXT    PRIMARY KEY,
    namespace    TEXT    NOT NULL,
    queue        TEXT    NOT NULL,
    kind         TEXT    NOT NULL,
    args         TEXT    NOT NULL DEFAULT '{}',
    unique_key   TEXT,
    state        TEXT    NOT NULL,
    priority     INTEGER NOT NULL,
    attempt      INTEGER NOT NULL,
    max_attempts INTEGER NOT NULL,
    scheduled_at INTEGER NOT NULL,
    attempted_at INTEGER,
    attempted_by TEXT,
    finalized_at INTEGER,
    last_error   TEXT,
    next_seq     INTEGER NOT NULL DEFAULT 1,
    version      INTEGER NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS job_unico ON job (namespace, unique_key)
    WHERE unique_key IS NOT NULL AND state NOT IN ('discarded', 'cancelled');

CREATE INDEX IF NOT EXISTS job_busca ON job (queue, priority, scheduled_at, job_id)
    WHERE state = 'available';

CREATE TABLE IF NOT EXISTS job_event (
    job_id      TEXT    NOT NULL REFERENCES job (job_id) ON DELETE CASCADE,
    seq         INTEGER NOT NULL,
    event_type  INTEGER NOT NULL,
    occurred_at INTEGER NOT NULL,
    payload     TEXT    NOT NULL,
    PRIMARY KEY (job_id, seq)
);
