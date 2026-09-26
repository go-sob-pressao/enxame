-- Plano de controle do pgcoord (Capítulo 24; ADR-003 e ADR-005).
--
-- coord_leader tem uma linha só. Adquirir a liderança é um UPDATE que só
-- acontece se o lease venceu ou já é seu; o termo cresce quando a
-- liderança troca de dono e é o fencing token do plano de controle: o
-- líder confere o próprio termo com FOR SHARE na mesma transação em que
-- grava um mapa novo. Quem está vivo vem de cluster_member (0009).
CREATE TABLE coord_leader (
    id         INT         PRIMARY KEY CHECK (id = 1),
    term       BIGINT      NOT NULL DEFAULT 0,
    node       TEXT,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT '-infinity'
);
INSERT INTO coord_leader (id) VALUES (1);

-- Um mapa partição→nó por época, gravado pelo líder do termo indicado.
CREATE TABLE coord_assignment (
    epoch      BIGINT      PRIMARY KEY,
    term       BIGINT      NOT NULL,
    owners     JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
