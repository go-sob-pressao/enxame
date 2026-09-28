// Package tresdamanha — Missão #7: três da manhã (Capítulo 30).
package tresdamanha

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// livro:inicio missao-07

// Config é a configuração do worker de relatórios, como ficou no deploy
// das 2h50. O plantão da semana anterior tinha reclamado que, depois de
// uma queda do worker, os relatórios órfãos levavam cinco minutos para
// voltar à fila; o prazo de resgate foi encurtado.
func Config(log *slog.Logger) enxame.WorkerConfig {
	return enxame.WorkerConfig{
		Queues:      map[string]int{"relatorios": 4},
		RescueAfter: 2 * time.Second, // era 5 min, o padrão
		Log:         log,
	}
}

// livro:fim missao-07

// Relatorio monta o relatório de fechamento de um cliente: consulta o
// ERP, que leva cerca de três segundos, e grava o resultado.
func Relatorio(log *slog.Logger, erp time.Duration) enxame.Handler {
	return func(ctx context.Context, j enxame.Job) error {
		var r struct {
			Cliente string `json:"cliente"`
		}
		if err := json.Unmarshal(j.Args, &r); err != nil {
			return enxame.Permanent(err)
		}
		inicio := time.Now()
		log.InfoContext(ctx, "consultando o ERP",
			slog.String("cliente", r.Cliente))
		select {
		case <-time.After(erp):
		case <-ctx.Done():
			log.WarnContext(ctx, "consulta ao ERP interrompida",
				slog.String("cliente", r.Cliente),
				slog.Duration("depois_de", time.Since(inicio)),
				slog.Any("causa", context.Cause(ctx)))
			return context.Cause(ctx)
		}
		log.InfoContext(ctx, "relatório gravado",
			slog.String("cliente", r.Cliente),
			slog.Duration("erp", time.Since(inicio)))
		return nil
	}
}
