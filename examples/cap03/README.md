# Capítulo 3 — Goroutines e o custo real

| Diretório | O que mostra | Como rodar |
|---|---|---|
| `preempcao/` | laço sem ponto de preempção | `go run ./examples/cap03/preempcao` e `GODEBUG=asyncpreemptoff=1 go run …` (trava: Ctrl-C) |
| `custo/` | criar goroutine, trocar de contexto, contenção | `go test -bench . -count 10 ./examples/cap03/custo \| benchstat -` |
| `paralelizar/` | quando mais goroutines deixam mais lento (`SomarQuadrados`, CPU) e quando deixam mais rápido (`Esperar`, espera) | `go test -bench . -count 10 ./examples/cap03/paralelizar` |
| `gomaxprocs/` | GOMAXPROCS ciente de cgroup (Go 1.25) | `docker run --cpus=2 …` (ver o main.go) |

Números do livro: medidos na máquina de referência (Plano Mestre §18, item 6),
com `-count=10` e `benchstat`. Os seus serão outros — a forma da curva, não.
