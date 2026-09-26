-- Quem está no cluster (Capítulo 23). Cada nó bate aqui; os instantes
-- são todos do relógio do banco, now(), e nunca do nó.
CREATE TABLE cluster_member (
    node       TEXT        PRIMARY KEY,
    addr       TEXT        NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),  -- muda a cada reinício
    beat_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
