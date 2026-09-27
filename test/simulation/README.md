# Simulação determinística

Cenários rodam com `-tags=simulation` (Capítulo 27):

    make sim                                   # seeds 1..200, Raft e Enxame
    make sim SEED=4                            # uma seed, com o diário
    go test -tags=simulation ./test/simulation/ -run Enxame \
        -args -seed=4 -particao=484            # o diário anota a partição
    go test -tags=simulation ./test/simulation/ -args -random-seeds=2000

Cada falha registra a seed e as violações em `failures/`, e a seed
reproduz a execução inteira, evento por evento.

- `TestSimulacaoRaft` — cinco nós Raft sobre a rede virtual.
- `TestSimulacaoEnxame` — três enxamed sobre o banco simulado: o
  Rebalanceador, o detector phi e o Distribute de produção; pausas,
  quedas, isolamento do banco e perda de instruções sorteados.
- `TestMesmaSeedMesmoRastro*` — a mesma seed dá o mesmo rastro.

Duas camadas, com papéis distintos:

- `testing/synctest` — teste unitário de código concorrente com relógio
  virtual, dentro de uma "bolha". Usado desde o Capítulo 11.
- `internal/simulation` — vários nós, falhas sorteadas por seed, o
  escalonador decidindo a ordem de tudo. É o que synctest não alcança.
