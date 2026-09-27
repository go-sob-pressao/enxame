# Decisões de arquitetura

Cada ADR nasce no capítulo indicado, com a alternativa **demonstrada** — não
apenas descrita — antes de ser rejeitada. As marcadas como *aceita* já estão
redigidas; as demais são registradas quando o capítulo correspondente é escrito.

| # | Decisão | Alternativa rejeitada | Cap. | Status |
|---|---|---|---|---|
| [001](0001-event-sourcing-do-historico.md) | Event sourcing do histórico de jobs e passos | Estado mutável com log de auditoria | 14 | aceita |
| [002](0002-workflow-por-passos-memoizados.md) | Workflow por passos memoizados | Replay determinístico com escalonador de corrotinas | 2 | aceita |
| [003](0003-coordinator-com-duas-implementacoes.md) | Coordenação atrás de um contrato, com duas implementações (Postgres e Raft) | Raft como única coordenação | 25 | aceita |
| [004](0004-postgresql-como-storage.md) | PostgreSQL como storage de produção | Storage engine próprio | 14 | aceita |
| [005](0005-fencing-token-no-banco.md) | Fencing token no banco, verificado com `FOR SHARE` | Lease apenas por timeout | 24 | aceita |
| [006](0006-lock-striped-por-chave.md) | Lock striped por chave (`job_id`, `ordering_key`) | Mutex por partição | 8 | aceita |
| [007](0007-enfileiramento-transacional.md) | Enfileiramento transacional (`InsertTx`, `PublishTx`) | Publicação direta em broker | 15 | aceita |
| [008](0008-binario-unico-papeis-modulares.md) | Binário único, papéis modulares; também embutível como biblioteca | Microsserviços desde o início | 2 | aceita |
| 009 | Simulação determinística | Apenas testes de integração | 27 | proposta |
| [010](0010-particoes-fixas.md) | Número de partições fixo (512) | Reparticionamento dinâmico | 26 | aceita |
