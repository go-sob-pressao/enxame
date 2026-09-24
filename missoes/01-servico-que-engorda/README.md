# Missão #1 — O serviço que engorda

> **Chamado #4471 — prioridade alta.** O `notificador` sobe com 40 MB e, em
> cinco dias, passa de 3 GB. Reiniciar resolve por uma semana. Os testes
> passam. Ninguém mudou nada de relevante no último mês.

Rode:

```bash
go test ./missoes/01-servico-que-engorda/ -run TestMissao
```

O teste está vermelho. Encontre e feche **os vazamentos** sem mudar o
comportamento: os demais testes do pacote precisam continuar verdes.

Critério: `go test -race ./missoes/01-servico-que-engorda/` verde.
Dicas e gabarito: no fechamento da Parte I do livro.
