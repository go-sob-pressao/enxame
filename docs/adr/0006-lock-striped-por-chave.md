# ADR 0006 — Lock striped por chave

- **Status:** aceita
- **Capítulo:** 8
- **Data:** 2026-09-24

## Contexto

O dono de uma partição executa, ao mesmo tempo, operações sobre muitos jobs:
concluir, registrar falha, cancelar, resgatar. Duas operações sobre o MESMO
job (ou a mesma `ordering_key`) precisam acontecer uma de cada vez; operações
sobre jobs diferentes, não. Uma partição tem milhares de jobs ativos.

## Opções consideradas

1. **Um `Mutex` por partição.** Correto e simples; serializa tudo.
2. **Um `Mutex` por chave, num map.** Paralelismo máximo; o map cresce com as
   chaves e precisa de outra trava para ser alterado, além de limpeza.
3. **Lock striped:** N mutexes fixos, a chave escolhe a faixa por hash.
   Memória constante; colisões entre chaves diferentes são raras e custam só
   uma espera curta.

## Decisão

Opção 3, com 64 faixas por partição, cada faixa ocupando uma linha de cache.
A opção 1 foi medida e rejeitada: no benchmark `BenchmarkTravaPorChave`
(64 goroutines, 20 µs por operação), o mutex por partição limita a vazão à de
uma goroutine; o striped escala com os núcleos. Números no Capítulo 8,
Custo Real #2, com máquina e método.

## Consequências

- **Melhor:** operações sobre jobs diferentes não se esperam.
- **Pior:** duas chaves na mesma faixa se serializam sem motivo; o número de
  faixas é um parâmetro a medir, não a adivinhar.
- **Mais difícil de mudar depois:** nada relevante — a interface `KeyLock`
  isola a escolha.
