package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Expr é uma expressão cron de cinco campos: minuto, hora, dia do mês,
// mês e dia da semana (0 = domingo). Cada campo aceita *, listas
// (1,15), intervalos (9-17) e passos (*/5, 0-30/10).
type Expr struct {
	minuto, hora, dia, mes, semana [60]bool
	diaLivre, semanaLivre          bool
}

var limites = [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}

// Parse interpreta a expressão.
func Parse(s string) (Expr, error) {
	campos := strings.Fields(s)
	if len(campos) != 5 {
		return Expr{}, fmt.Errorf("cron %q: esperava 5 campos", s)
	}
	var e Expr
	alvos := []*[60]bool{&e.minuto, &e.hora, &e.dia, &e.mes, &e.semana}
	for i, c := range campos {
		if err := campo(c, limites[i], alvos[i]); err != nil {
			return Expr{}, fmt.Errorf("cron %q: %w", s, err)
		}
	}
	e.diaLivre, e.semanaLivre = campos[2] == "*", campos[4] == "*"
	return e, nil
}

func campo(s string, lim [2]int, alvo *[60]bool) error {
	for parte := range strings.SplitSeq(s, ",") {
		faixa, passo := parte, 1
		if a, b, ok := strings.Cut(parte, "/"); ok {
			n, err := strconv.Atoi(b)
			if err != nil || n <= 0 {
				return fmt.Errorf("passo inválido em %q", parte)
			}
			faixa, passo = a, n
		}
		ini, fim := lim[0], lim[1]
		if faixa != "*" {
			a, b, intervalo := strings.Cut(faixa, "-")
			var err error
			if ini, err = strconv.Atoi(a); err != nil {
				return fmt.Errorf("valor inválido em %q", parte)
			}
			fim = ini
			if intervalo {
				if fim, err = strconv.Atoi(b); err != nil {
					return fmt.Errorf("valor inválido em %q", parte)
				}
			}
		}
		if ini < lim[0] || fim > lim[1] || ini > fim {
			return fmt.Errorf("%q fora de %d-%d", parte, lim[0], lim[1])
		}
		for v := ini; v <= fim; v += passo {
			alvo[v] = true
		}
	}
	return nil
}

// Next devolve o primeiro minuto depois de t, no fuso loc, que satisfaz
// a expressão. Procura por até cinco anos; zero se não houver.
func (e Expr) Next(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc).Truncate(time.Minute).Add(time.Minute)
	for limite := t.AddDate(5, 0, 0); t.Before(limite); {
		if !e.mes[int(t.Month())] {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !e.casaDia(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0,
				loc)
			continue
		}
		if !e.hora[t.Hour()] {
			t = t.Truncate(time.Hour).Add(time.Hour)
			continue
		}
		if !e.minuto[t.Minute()] {
			t = t.Add(time.Minute)
			continue
		}
		return t
	}
	return time.Time{}
}

// casaDia segue a regra do cron: se os dois campos de dia forem
// restritos, basta um deles casar.
func (e Expr) casaDia(t time.Time) bool {
	d, s := e.dia[t.Day()], e.semana[int(t.Weekday())]
	switch {
	case e.diaLivre && e.semanaLivre:
		return true
	case e.diaLivre:
		return s
	case e.semanaLivre:
		return d
	}
	return d || s
}
