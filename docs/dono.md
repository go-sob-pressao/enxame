# Dono do Enxame

Um sistema sem dono declarado já está quebrado — ele só ainda não
sabe disso (Regra 32).

| | |
|---|---|
| Dono | time Plataforma de Jobs |
| Plantão | escala semanal do time, com substituto nomeado |
| Onde chamar | canal `#enxame-plantao`; a página vem do Alertmanager |
| SLOs | `docs/slo.md` |
| Runbooks | `docs/runbook/` — um por alarme |
| Postmortems | `docs/postmortem/`, pelo modelo |
| Decide congelar versões | o dono, pela política de `docs/slo.md` |

O dono responde pelo Enxame, não pelo código de cada aplicação que o
usa: o handler que falha é do time da aplicação, e a fila que não anda
porque o handler falha também. O runbook de cada alarme diz quem
chamar quando a causa está do outro lado.

Troque este arquivo quando o Enxame for para o seu time: o nome do
time e do canal são os de um exemplo.
