# 01 — Primeiro job

Um handler tipado, uma fila, um worker no mesmo processo (modo biblioteca).

```go
type EnviarEmail struct{ Para, Assunto string }

func (EnviarEmail) Kind() string { return "email.enviar" }

w := enxame.NewWorker(db, enxame.Queue("padrao", 20))
enxame.Handle(w, func(ctx context.Context, j *job.Job[EnviarEmail]) error {
    return mailer.Send(ctx, j.Args.Para, j.Args.Assunto, j.IdempotencyKey())
})
go w.Run(ctx)

client.Insert(ctx, EnviarEmail{Para: "ana@exemplo.com", Assunto: "Bem-vinda"})
```

Nasce no **Capítulo 6** (pool de workers) e ganha durabilidade no **Capítulo 14**.
