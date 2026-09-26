# Missão #3 — O pagamento em dobro

**Chamado #6071, prioridade alta.** Três clientes reclamaram de cobrança em
dobro na mesma semana. O gateway confirma: duas cobranças para o mesmo
pedido, com minutos de diferença. O time de pagamentos jura que o job de
cobrança roda uma vez por job; o de checkout jura que o pedido é gravado uma
vez só.

```bash
make up
export ENXAME_DB_DSN=postgres://…   # o de make up
go test -count=1 -run TestMissao ./missoes/03-pagamento-em-dobro/
```

Encontre a janela e feche-a **sem mudar o teste nem o handler de cobrança**.

**Critério:** `TestMissao` verde, e `go test ./missoes/03-pagamento-em-dobro/`
verde. Dicas em três níveis e o gabarito comentado estão no fechamento da
Parte III.
