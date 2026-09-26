// Package perdida mostra a atualização perdida em READ COMMITTED, e as
// três formas de evitá-la no PostgreSQL.
package perdida

// Esquema cria a conta do exemplo, com saldo 100.
const Esquema = `CREATE TABLE conta (
    id      INT    PRIMARY KEY,
    saldo   INT    NOT NULL,
    version BIGINT NOT NULL DEFAULT 1
);
INSERT INTO conta (id, saldo) VALUES (1, 100);`
