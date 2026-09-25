package workflow

import (
	"encoding/json"
	"time"
)

// Sleep suspende o workflow por d. A hora de acordar é gravada na
// primeira execução; os replays seguintes comparam o relógio com ela, e
// não com d — senão cada replay adiaria o despertar.
func Sleep(c *Context, name string, d time.Duration) error {
	r, gravado, err := c.proximo(name, KindSleep)
	if err != nil {
		return err
	}
	if !gravado {
		r.WakeAt = c.clock().Add(d)
		if err := c.hist.Append(c.ctx, r); err != nil {
			return err
		}
	}
	if c.clock().Before(r.WakeAt) {
		return &SuspendedError{Until: r.WakeAt}
	}
	return nil
}

// SideEffect grava o valor que fn devolve na primeira execução, e o
// devolve em cada replay. É para o que muda a cada chamada — um número
// aleatório, um UUID — e não tem efeito fora do processo.
func SideEffect[T any](
	c *Context,
	name string,
	fn func() T,
) (T, error) {
	var zero T
	r, gravado, err := c.proximo(name, KindSideEffect)
	if err != nil {
		return zero, err
	}
	if gravado {
		var v T
		err := json.Unmarshal(r.Output, &v)
		return v, err
	}
	v := fn()
	if r.Output, err = json.Marshal(v); err != nil {
		return zero, err
	}
	return v, c.hist.Append(c.ctx, r)
}

// Now devolve a hora da primeira execução deste ponto do workflow, e a
// mesma hora em cada replay.
func Now(c *Context) (time.Time, error) {
	return SideEffect(c, "workflow.Now", c.clock)
}
