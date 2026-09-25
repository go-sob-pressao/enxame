# Missão #2 — O teste que falha às terças

> **Chamado #5120 — prioridade média.** O `TestMissao` do agrupador de
> auditoria falha na CI de vez em quando. O time aprendeu a clicar em
> "rodar de novo". Alguém notou que as falhas se concentram às terças, dia
> de deploy, quando a CI está mais carregada.

Rode:

```bash
go test -race -count=2000 -run TestMissao ./missoes/02-teste-que-falha-as-tercas/
```

Algumas das duas mil execuções falham. Torne o `TestMissao` **determinístico**:
ele deve continuar afirmando a mesma coisa — dez eventos seguidos formam um
único lote de dez — sem aumentar o intervalo nem o tempo de espera, e sem
mudar o `agrupador.go`.

Critério: o comando acima verde, duas mil vezes em duas mil; `go test -race
./missoes/02-teste-que-falha-as-tercas/` verde. Dicas e gabarito: no
fechamento da Parte II do livro.
