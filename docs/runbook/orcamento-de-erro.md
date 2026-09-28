# Orçamento de erro queimando

**Alarmes:** `OrcamentoQueimandoRapido` (página: 2% do orçamento do
mês em 1 hora) e `OrcamentoQueimandoDevagar` (ticket: 5% em 6
horas), para cada SLO de `docs/slo.md`.

**Impacto:** o rótulo `slo` diz qual. No ritmo rápido, o orçamento do
mês inteiro acaba em dois dias.

## Confirmar

O painel "Enxame", linha SLO, mostra a taxa de erro de cada SLO nas
janelas de 5 min e 1 h. As duas acima do limiar é o alarme de
verdade; só a curta, uma rajada que já passou.

## Agir

1. **`slo="enfileirar"`:** a API responde 5xx. Os logs com
   `"status":5` e o trace de uma requisição falha dizem qual erro. Se
   é o banco, é o banco; se é `503` do descarte de carga, é excesso.
2. **`slo="latencia"`:** o p99 do `POST /v1/jobs`. O flight recorder
   (`-voo-limiar`) guarda o trace de uma requisição lenta.
3. **`slo="pontualidade"`:** jobs começando tarde:
   `fila-crescendo.md`.
4. Uma versão nova nas últimas horas? `rollback-versao.md` primeiro,
   investigação depois.

## Depois

Orçamento do mês esgotado: vale a política de `docs/slo.md` — as
versões novas param, exceto as que consertam confiabilidade.

## Escalar

O dono do sistema.
