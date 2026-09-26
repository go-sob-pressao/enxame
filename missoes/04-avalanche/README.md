# Missão #4 — A avalanche

**Chamado #7730, prioridade alta.** O endpoint de webhooks de um cliente ficou
fora do ar por uma hora. Quando voltou, caiu de novo em segundos — e de novo,
e de novo. O cliente diz que o Enxame o está atacando; o time diz que só está
repetindo o que falhou, com backoff exponencial.

```bash
go test -count=1 -v -run TestMissao ./missoes/04-avalanche/
```

A simulação é determinística: 2.000 entregas, uma hora de queda, um endpoint
que aguenta 100 requisições por segundo e cai por mais um minuto quando recebe
mais do que o triplo disso num segundo. Estanque a avalanche mudando **só**
`retry.go`.

**Critério:** `TestMissao` verde — todas as entregas em até duas horas, sem
nenhuma recaída. Dicas em três níveis e o gabarito comentado estão no
fechamento da Parte IV.
