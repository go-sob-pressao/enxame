# ADR 0001 — Event sourcing do histórico de jobs e passos

- **Status:** aceita
- **Capítulo:** 14
- **Data:** 2026-09-25

## Contexto

Todo incidente com um job termina na mesma pergunta: o que aconteceu com
ele, em que ordem, e por quê. O Enxame precisa responder a ela depois de um
`kill -9`, de um deploy no meio de uma tentativa, de um resgate. E o estado
corrente — `available`, `running`, `completed` — precisa ser consultado a
cada busca, sem custo proporcional à idade do job.

## Opções consideradas

1. **Estado mutável com log de auditoria.** A tabela `job` é a verdade; cada
   mudança também grava uma linha de auditoria. Simples e familiar.
2. **Event sourcing.** O histórico `job_event`, só de acréscimo, é a
   verdade; `job` é uma projeção dele, atualizada na mesma transação. O
   domínio só muda o estado devolvendo eventos, e `Apply` é a única função
   que os projeta.

## Decisão

Opção 2. A opção 1 foi demonstrada e rejeitada em
`examples/cap14/auditoria` (`-tags defeito`): um caminho de código escrito
depois — um resgate — muda o estado sem registrar nada, e a auditoria
passa a contar outra história. Nada no sistema percebe, porque o estado
não depende da auditoria. Na opção 2, esse caminho não existe: um resgate
que não devolve evento não muda o estado. E a divergência entre projeção e
histórico passa a ser verificável — o Experimento 14.1 reconstrói cada job
depois de um `kill -9` e compara.

O custo do replay foi medido (Capítulo 14, Custo Real #3): 160 ms e 70 MiB
para 100.000 eventos, contra 0,4 ms da leitura da projeção. Por isso a
projeção é mantida sempre em dia, na mesma transação do evento, e é ela
que a busca lê.

## Consequências

- **Melhor:** o histórico responde "o que aconteceu" sem depender da
  disciplina de quem escreve o próximo caminho de código; a projeção pode
  ser conferida e reconstruída.
- **Pior:** toda transição escreve duas tabelas; o histórico cresce sem
  limite e vai precisar de retenção (Parte VII).
- **Mais difícil de mudar depois:** o formato dos eventos. Um evento
  gravado é para sempre; mudar o significado de um tipo exige uma versão
  nova do tipo, não uma edição.
