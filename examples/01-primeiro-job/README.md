# 01 — Primeiro job

Um handler, uma fila, um worker no mesmo processo — o modo biblioteca, só com
`pkg/enxame`.

```go
client := enxame.New(db, "minha-app")
client.Insert(ctx, EnviarEmail{Para: "ana@exemplo.com", Assunto: "Bem-vinda"})

w := client.NewWorker(enxame.WorkerConfig{})
w.Handle("email.enviar", func(ctx context.Context, j enxame.Job) error {
    var e EnviarEmail
    if err := json.Unmarshal(j.Args, &e); err != nil {
        return enxame.Permanent(err)
    }
    return mailer.Send(ctx, e.Para, e.Assunto, job.IdempotencyKey(ctx))
})
w.Run(ctx)
```

```bash
make up
export ENXAME_DB_DSN=postgres://postgres:enxame@localhost:5432/enxame?sslmode=disable
go run ./examples/01-primeiro-job
```

Nasce no **Capítulo 6** (pool de workers), ganha durabilidade no **Capítulo
14**, a chave de idempotência no **16** e o worker público no **17**.
