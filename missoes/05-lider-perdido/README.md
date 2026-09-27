# Missão #5 — O líder perdido

**Chamado #8104, prioridade crítica.** Às 3h12, o nó líder de um cluster
de três caiu, no meio de uma rajada de lembretes agendados para as 3h13.
O failover funcionou: os outros dois assumiram as partições dele em
segundos, e o painel de partições ficou verde. Às 9h, a central de
atendimento reclamou de dezenas de lembretes que nunca saíram. Estão
todos no banco, em `scheduled`, com a hora vencida há seis horas. Nenhum
erro nos logs. Os lembretes dos outros nós saíram todos.

```bash
make up
go test -count=1 -v -run TestMissao ./missoes/05-lider-perdido/
```

O teste sobe três nós no mesmo processo, sobre um banco novo, agenda 120
jobs para dali a um segundo e derruba o líder. Em 15 segundos, os 120
precisam estar disponíveis. Na tag `missao-05`, 36 continuam agendados
para sempre. Conserte mudando **só** `promotor.go`.

**Critério:** `TestMissao` verde, e nenhum job promovido por um nó que não
é o dono da partição dele. Dicas em três níveis e o gabarito comentado
estão no fechamento da Parte V.
