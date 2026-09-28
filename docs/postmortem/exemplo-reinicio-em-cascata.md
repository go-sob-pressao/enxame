# Postmortem: o banco congelou por 30 segundos e os três nós reiniciaram

Exemplo do Capítulo 32: um caso composto, reproduzido no laboratório
de `examples/cap32/incidente`. Os números e horários são os da
reprodução.

| | |
|---|---|
| Duração | 30 s de banco parado; 36 s até os três nós prontos de vez |
| Impacto | 3.390 requisições falharam e foram repetidas pelos clientes; nenhum job aceito se perdeu; 4 rodaram duas vezes |
| Orçamento de erro gasto | `enfileirar`: as 3.390 |
| Detecção | clientes recebendo `EOF` |

## Linha do tempo

A partir do instante em que o banco parou.

| t | O que aconteceu |
|---|---|
| 0 s | O PostgreSQL para de responder: congelado, sem cair. |
| 2–3 s | A liveness dos três nós, que chama o `/statusz`, estoura o prazo de 1 s. Com `failureThreshold: 1`, o kubelet decide reiniciar os três. |
| 3–30 s | Depois do `preStop` de 5 s, o `SIGTERM`: cada nó para de aceitar e fecha as conexões, e espera as requisições presas no banco. Os clientes recebem `EOF` — 3.267 vezes — e repetem. |
| 30 s | O banco volta. As requisições presas terminam, os processos velhos saem e os novos começam, no mesmo segundo. |
| 32 s | Os três prontos. |
| 33–35 s | Nenhum pronto: as milhares de repetições acumuladas chegam juntas, a readiness estoura o prazo, e o Kubernetes tira os três do Service. |
| 36 s | Os três prontos de vez. 4 tentativas que estavam em curso rodam de novo. |

## Causa

A liveness perguntava se o banco respondia. O banco é o mesmo para os
três nós: quando ele parou, as três probes falharam juntas, e o
reinício — que existe para tirar um processo travado do buraco — foi
aplicado a três processos saudáveis, ao mesmo tempo, por um problema
que reiniciar não resolve. O reinício nem chegou a acontecer antes de
o banco voltar: o desligamento esperou o banco. O que ele fez foi
fechar as portas durante o congelamento. O `failureThreshold: 1`
tinha sido escolhido para matar mais depressa um pod travado.

## O que funcionou

- Nenhum job aceito se perdeu; os efeitos repetidos eram idempotentes
  pela chave.
- Os nós novos esperaram o banco em vez de sair com erro
  (`migrarQuandoDer`).

## O que não funcionou

- A probe transformou a falha de uma dependência em desligamento de
  todos: em vez de esperar o banco, os clientes receberam `EOF`.
- As repetições dos clientes, sem espera crescente, chegaram todas no
  mesmo segundo e derrubaram a readiness dos nós recém-subidos.

## Onde tivemos sorte

- O banco voltou em 30 s. Numa queda de minutos, os contêineres novos
  subiriam sem banco, a startupProbe os mataria depois de 60 s, e o
  `CrashLoopBackOff` dobraria a espera até 5 minutos: o Enxame
  voltaria muito depois do banco.

## Ações

| Ação | Tipo | Dono | Prazo |
|---|---|---|---|
| Liveness no `/healthz`, que não consulta nada; três falhas de 10 s | prevenir | Plataforma de Jobs | feito (cap-32) |
| Readiness sem o banco: ele fora deixa todos os nós igualmente inúteis | prevenir | Plataforma de Jobs | feito (cap-32) |
| O estado do banco no `/statusz`, para gente, fora das probes | detectar | Plataforma de Jobs | feito (cap-32) |
| Clientes repetindo com espera crescente e jitter (Cap. 20) | mitigar | times das aplicações | próxima versão de cada cliente |

Com as probes corrigidas, o mesmo congelamento de 30 s: nenhum
reinício, 128 tentativas falhas — todas por prazo do cliente, enquanto
esperavam o banco —, 2 jobs repetidos cuja conclusão não chegou ao
banco congelado, e o Enxame andando no segundo em que o banco voltou.
