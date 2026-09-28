# Missão #6 — O p99 que subiu oito vezes

**Chamado #9127, prioridade alta.** Na terça, o p99 do `POST /v1/jobs`
subiu oito vezes, de um dia para o outro, e a CPU dos nós da API dobrou.
O único deploy da segunda foi um commit de configuração: a API passou a
aceitar só os 400 kinds que a aplicação conhece (`-kinds`), para barrar
jobs com nome digitado errado. O commit "não mexeu em nada" do caminho
quente. Nenhum erro nos logs.

```bash
make up
git checkout missao-06-p99-oito-vezes
go test -count=1 -v -run TestMissao ./missoes/06-p99-oito-vezes/
```

O teste monta a API com os 400 kinds, confere que um kind da lista é
aceito e um de fora é recusado, e mede as alocações e o tempo de CPU
por requisição, com e sem a lista. Num notebook, a latência de cada
requisição é quase toda espera pelo banco, e o p99 não mostra o
defeito; nos nós do chamado, que já rodavam perto do limite de CPU,
mostrou. Ache o custo com o `pprof` — não pela leitura do
diff — e conserte mudando **só** `internal/transport/http/kinds.go`.

**Critério:** `TestMissao` verde: no máximo 200 alocações por
requisição, e a lista continua valendo. Dicas em três níveis e o
gabarito comentado estão no fechamento da Parte VI.
