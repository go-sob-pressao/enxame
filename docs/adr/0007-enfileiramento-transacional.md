# ADR 0007 — Enfileiramento transacional

- **Status:** aceita
- **Capítulo:** 15
- **Data:** 2026-09-25

## Contexto

A aplicação do leitor grava dados de negócio — um pedido — e precisa
disparar trabalho a partir deles — a cobrança. Escrever os dois em
sistemas diferentes (dual-write) deixa uma janela em que um existe e o
outro não; uma falha nela perde trabalho (pedido sem cobrança) ou inventa
trabalho (cobrança sem pedido).

## Opções consideradas

1. **Publicação direta num broker**, antes ou depois do `COMMIT`.
2. **Outbox próprio da aplicação**: uma tabela de saída na transação do
   pedido, e um despachante escrito pela aplicação.
3. **`InsertTx` e `PublishTx`**: o cliente do Enxame grava o job (ou a
   mensagem de webhook e o job que a distribui) na transação da própria
   aplicação.

## Decisão

Opção 3. A opção 1 foi demonstrada e rejeitada em
`examples/cap15/dualwrite` (`-tags defeito`), nas duas ordens: publicar
depois do `COMMIT` perde a cobrança numa queda entre as escritas;
publicar antes entrega ao consumidor um pedido que ele não encontra, e
que pode nunca existir. A opção 2 é a correta para quem não compartilha
o banco com o Enxame, e continua disponível pelo `Insert`.

## Consequências

- **Melhor:** o dado e o trabalho existem juntos ou não existem; a
  aplicação não escreve código de despacho. A mesma transação atômica é
  a que o próprio Enxame usa para avançar workflows (`AppendStep`).
- **Pior:** exige que a aplicação use o mesmo PostgreSQL e o `pgx`
  (`database/sql` precisaria de um adaptador); a transação da aplicação
  fica mais longa. No Custo Real #4, com oito goroutines, criar pedido e
  job numa transação custou 1,06 ms, contra 1,62 ms em duas transações e
  0,32 ms com a mensagem num canal em memória — a diferença é de idas e
  voltas ao banco, não de `COMMIT`s.
- **Mais difícil de mudar depois:** o esquema do Enxame passa a morar no
  banco da aplicação; migrações dele são migrações dela (Capítulo 17).
