-- Propriedade de partição e fencing token (ADR-005, ADR-010).
--
-- range_id é incrementado a cada aquisição. Toda transação do dono começa com
--
--     SELECT range_id FROM partition_lease WHERE partition_id = $1 FOR SHARE;
--
-- e aborta se o valor lido diferir do range_id que o nó carrega. O FOR SHARE
-- é indispensável: sem ele, em READ COMMITTED, um nó zumbi valida o range_id,
-- o novo dono comita a aquisição, e a escrita do zumbi passa assim mesmo.
-- Com ele, a aquisição (UPDATE) espera a transação do zumbi terminar, e toda
-- escrita posterior do zumbi enxerga o range_id novo e é rejeitada.
CREATE TABLE partition_lease (
    partition_id     INT         PRIMARY KEY CHECK (partition_id >= 0),
    range_id         BIGINT      NOT NULL DEFAULT 0,
    owner            TEXT,
    lease_expires_at TIMESTAMPTZ
);

-- NumPartitions é fixo na criação do cluster (ADR-010).
INSERT INTO partition_lease (partition_id)
SELECT g FROM generate_series(0, 511) AS g;
