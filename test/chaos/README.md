# Engenharia do caos

Experimentos contra o cluster real (Capítulo 28): três `enxamed`, um
worker remoto em cada, o PostgreSQL de `make up`. Cada nó fala com o
banco por um proxy do laboratório, que corta, atrasa ou deixa passar.

    make up
    make chaos                       # todos os experimentos
    make chaos EXP=ParticaoDoLider    # um só

Cada experimento declara, no comentário, hipótese, injeção, raio de
alcance e o que refuta a hipótese. As séries de amostras vão para
`testdata/`. Toda falha encontrada aqui vira um teste de regressão — na
simulação (`test/simulation/`) quando a simulação alcança o defeito, na
integração quando o defeito mora no SQL ou no processo.
