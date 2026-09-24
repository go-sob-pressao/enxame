# ADR 0003 — Coordenação atrás de um contrato, com duas implementações

- **Status:** aceita
- **Capítulo:** 23 a 26 (contrato) · 24 (pgcoord) · 25 (raftcoord)
- **Data:** 2026-09-24

## Contexto

O cluster precisa responder, a todo instante: quem está vivo, quem é líder,
e qual nó é dono de cada partição. A resposta clássica é Raft. Mas o
plano de dados já vive num PostgreSQL linearizável, e o fencing por
`range_id` no banco já garante, sozinho, que só um nó escreve em cada
partição (ADR-005). Um leitor atento pergunta, com razão: *se o banco já
coordena, por que Raft?* Sem resposta honesta, o capítulo mais importante do
livro vira enfeite — e a lição "quando não usar consenso" se volta contra ele.

## Opções consideradas

1. **Só Raft.** Didático, mas injustificável em produção quando há um banco
   transacional à disposição.
2. **Só Postgres.** Correto e simples; elimina o capítulo de consenso.
3. **Contrato `coordinator.Coordinator` com duas implementações**, ambas
   aprovadas pela mesma suíte de conformidade e pela simulação:
   - `pgcoord` — lease e eleição sobre linhas de controle no Postgres, com
     fencing. **Padrão de produção.**
   - `raftcoord` — o Raft implementado no Capítulo 25, replicando o mapa
     partição→nó sem depender do banco para o plano de controle.

## Decisão

Opção 3.

O Capítulo 24 constrói `pgcoord` e demonstra que ele basta. O Capítulo 25
constrói Raft do zero e `raftcoord` sobre ele, e **mede os dois** na mesma
bancada: tempo de failover, carga sobre o banco, comportamento durante
failover do próprio Postgres. A escolha fica com o operador, documentada.

## Consequências

- **Melhora:** Raft deixa de ser enfeite e vira a lição mais madura do livro —
  implementar consenso, medir contra a alternativa simples, decidir com
  números. É também a demonstração mais forte do Capítulo 12: uma suíte, duas
  implementações.
- **Mantém:** Raft completo — eleição, replicação, segurança, snapshot — e o
  estado replicado pequeno (512 entradas).
- **Piora:** duas implementações para manter. Mitigado pela suíte comum e
  pelo tamanho reduzido de `pgcoord`.
- **Regra de arquitetura:** `internal/cluster/coordinator` não importa as
  implementações; `internal/cluster/raft` não importa store nem engine.
  Verificado por `tools/archcheck`.
