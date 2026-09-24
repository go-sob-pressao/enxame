# ADR 0008 — Binário único, papéis modulares, embutível como biblioteca

- **Status:** aceita
- **Capítulo:** 2 (registro) · 17 (modo biblioteca) · 18 (modo servidor)
- **Data:** 2026-09-24

## Contexto

O Enxame tem cinco papéis: api, worker, scheduler (dono de partições),
delivery (webhooks) e cluster (coordenação). E dois públicos: a aplicação que
quer jobs em segundo plano contra o próprio Postgres, sem operar mais nada, e a
equipe de plataforma que quer um serviço compartilhado em cluster.

Separar os papéis em serviços desde o início cobraria, antes de o leitor
entender o motor, o preço de uma rede entre cada par de componentes:
serialização, timeout, retry, versionamento de protocolo e implantação de
cinco artefatos. O livro só pode ensinar essas coisas quando o domínio
exigir — na Parte IV.

## Opções consideradas

1. **Microsserviços desde o início.** Um processo por papel, gRPC entre eles.
   Escala independente; cinco pipelines, cinco imagens, rede em toda chamada.
2. **Binário único, papéis ativados por configuração.** Um `enxamed` que
   hospeda qualquer combinação de papéis; os papéis falam por interfaces Go
   quando estão no mesmo processo e por gRPC quando não estão.
3. **Somente biblioteca.** Tudo dentro da aplicação do leitor; sem servidor.
   Simples, mas não ensina distribuição nem serve como plataforma
   compartilhada.

## Decisão

Opção 2, com o motor também **embutível**: `pkg/enxame` expõe o cliente e o
worker para a aplicação do leitor rodar no próprio processo (modo biblioteca,
Partes I a III). O `enxamed` usa o mesmo motor em modo servidor (Parte IV em
diante).

## Consequências

- **Melhor:** um artefato para construir, testar e implantar; o leitor adota o
  modo biblioteca sem operar um servidor; a fronteira entre papéis existe desde
  o Cap. 2 como interface, e vira rede só quando o livro chega à rede.
- **Pior:** papéis não escalam de forma independente sem configuração
  cuidadosa; um defeito de memória num papel derruba os outros no mesmo
  processo.
- **Mais difícil de mudar depois:** separar um papel em serviço próprio exige
  que a interface entre papéis continue estável — por isso ela é declarada no
  consumidor (Cap. 12) e testada por suíte de contrato.
