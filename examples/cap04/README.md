# Capítulo 4 — Channels como estrutura de dados

| Diretório | O que mostra |
|---|---|
| `fechamento/` | vários produtores e um único fechamento; `-tags defeito` reproduz o `send on closed channel` do enigma |
| `nilchan/` | `nil` channel no `select` para desligar um caso |
| `contador/` | o mesmo contador com channel e com Mutex (Anti-Pattern #1), com benchmark |
| `operacoes/` | a tabela de operações em channel aberto, fechado e nil, provada por teste |
