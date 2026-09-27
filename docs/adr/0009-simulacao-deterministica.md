# ADR 0009 — Simulação determinística dirigida por seed

- **Status:** aceita
- **Capítulo:** 27
- **Data:** 2026-09-27

## Contexto

As falhas que importam num sistema distribuído dependem de intercalações
raras — esta instrução se perdeu, aquele nó parou naquele instante, o
mapa mudou duas vezes durante uma drenagem. Os testes de integração e o
Teste de Realidade #3 exercitam os cenários que o autor imaginou; o M5
passava em todos eles e perdia jobs em metade dos cenários sorteados.

## Opções consideradas

1. **Só testes de integração**, contra o Postgres real, com cenários
   roteirizados.
2. **Simulação determinística**: o sistema conduzido por um escalonador
   de eventos em tempo virtual, com toda fonte de acaso vinda de uma
   seed, e propriedades conferidas em cada cenário.
3. **Gravar e reproduzir** execuções reais.

## Decisão

Opção 2, com a fronteira no SQL: dentro da simulação roda o código de
produção que toma decisões — o `Rebalanceador`, o detector phi, o
`Distribute`, as `Posses`, o nó Raft —, e cada instrução SQL que ele
emite é substituída por um modelo, em `internal/simulation/banco`, com a
semântica da instrução real. O que roda na simulação não cria goroutine,
não lê o relógio do sistema e não itera um `map` onde a ordem importa.

## Consequências

- O código de produção paga a disciplina: o `Rebalanceador` virou uma
  máquina de passos, com lease, contagem e relógio injetados.
- Cada modelo de instrução é uma segunda implementação a manter; o que
  o modelo esquecer, a simulação não vê. Os testes de integração
  continuam sendo quem confere o SQL.
- Mil cenários do Enxame por minuto, numa CPU; a varredura noturna roda
  8.000 seeds de cada simulação e publica a seed que falhar.
- Uma falha é reproduzível para sempre: a seed a refaz, evento por
  evento, e o teste de rastro garante que continue assim.
