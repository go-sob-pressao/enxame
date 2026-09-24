# Decisões de arquitetura

Cada ADR nasce no capítulo indicado, com a alternativa **demonstrada** — não
apenas descrita — antes de ser rejeitada. As marcadas como *aceita* já estão
redigidas; as demais são registradas quando o capítulo correspondente é escrito.

| # | Decisão | Alternativa rejeitada | Cap. | Status |
|---|---|---|---|---|
| 001 | Event sourcing do histórico de jobs e passos | Estado mutável com log de auditoria | 14 | proposta |
| [002](0002-workflow-por-passos-memoizados.md) | Workflow por passos memoizados | Replay determinístico com escalonador de corrotinas | 2 | aceita |
| [003](0003-coordinator-com-duas-implementacoes.md) | Coordenação atrás de um contrato, com duas implementações (Postgres e Raft) | Raft como única coordenação | 25 | aceita |
| 004 | PostgreSQL como storage de produção | Storage engine próprio | 14 | proposta |
| [005](0005-fencing-token-no-banco.md) | Fencing token no banco, verificado com `FOR SHARE` | Lease apenas por timeout | 24 | aceita |
| 006 | Lock striped por chave (`job_id`, `ordering_key`) | Mutex por partição | 8 | proposta |
| 007 | Enfileiramento transacional (`InsertTx`, `PublishTx`) | Publicação direta em broker | 15 | proposta |
| 008 | Binário único, papéis modulares; também embutível como biblioteca | Microsserviços desde o início | 2 | proposta |
| 009 | Simulação determinística | Apenas testes de integração | 27 | proposta |
| 010 | Número de partições fixo (512) | Reparticionamento dinâmico | 26 | proposta |
