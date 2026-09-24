-- Webhooks: a aplicação publica uma mensagem; o Enxame a entrega a cada
-- endpoint inscrito, assinada, com retry, ordem por endpoint e registro de
-- cada tentativa.
CREATE TABLE webhook_endpoint (
    endpoint_id     UUID        PRIMARY KEY DEFAULT uuidv7(),
    namespace       TEXT        NOT NULL,
    url             TEXT        NOT NULL CHECK (url ~ '^https?://'),
    description     TEXT        NOT NULL DEFAULT '',
    event_types     TEXT[]      NOT NULL DEFAULT '{}',   -- vazio = todos os tipos
    secret_ref      TEXT        NOT NULL,   -- referência ao segredo; nunca o segredo em claro
    rate_limit_rps  INT         CHECK (rate_limit_rps IS NULL OR rate_limit_rps > 0),
    disabled_at     TIMESTAMPTZ,
    disabled_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX webhook_endpoint_ativo ON webhook_endpoint (namespace) WHERE disabled_at IS NULL;

-- A mensagem é o outbox: Publish/PublishTx a grava na mesma transação dos
-- dados de negócio, junto com um job 'webhook.fanout'. O fan-out cria um job
-- 'webhook.deliver' por endpoint, com ordering_key = endpoint_id.
CREATE TABLE webhook_message (
    message_id      UUID        PRIMARY KEY DEFAULT uuidv7(),
    namespace       TEXT        NOT NULL,
    event_type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    idempotency_key TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX webhook_message_idempotente ON webhook_message (namespace, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Uma linha por tentativa, para diagnóstico e para o portal do consumidor.
CREATE TABLE webhook_attempt (
    message_id       UUID        NOT NULL REFERENCES webhook_message (message_id) ON DELETE CASCADE,
    endpoint_id      UUID        NOT NULL REFERENCES webhook_endpoint (endpoint_id) ON DELETE CASCADE,
    attempt          INT         NOT NULL CHECK (attempt > 0),
    job_id           UUID        NOT NULL,
    status_code      INT,
    duration_ms      INT         NOT NULL CHECK (duration_ms >= 0),
    error            TEXT,
    response_excerpt TEXT,       -- primeiros bytes da resposta, truncados
    attempted_at     TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (message_id, endpoint_id, attempt)
);

CREATE INDEX webhook_attempt_por_endpoint ON webhook_attempt (endpoint_id, attempted_at DESC);
