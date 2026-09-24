# Simulação determinística

Cenários rodam com `-tags=simulation` e entram no **Capítulo 27**. Até lá,
`make sim` informa SKIP — nunca um verde vazio.

Cada falha registra a seed em `failures/`, tornando o bug reproduzível
indefinidamente:

    go test -tags=simulation ./test/simulation/... -run TestParticaoDuranteRebalanceamento -args -seed=8371

Duas camadas, com papéis distintos:

- `testing/synctest` (stdlib, Go 1.25+) — teste unitário de código concorrente
  com relógio virtual, dentro de uma "bolha". Usado desde o Capítulo 11.
- `internal/simulation` — vários nós, rede virtual com partição, relógios
  divergentes entre nós e falha de disco injetada. É o que synctest não alcança.
