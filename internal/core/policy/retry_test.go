package policy_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/policy"
)

func TestRetry(t *testing.T) {
	r := policy.Retry{Base: time.Second, Max: time.Minute}
	for _, c := range []struct {
		tentativa int
		teto      time.Duration
	}{
		{1, time.Second}, {2, 2 * time.Second}, {3, 4 * time.Second},
		{6, 32 * time.Second}, {7, time.Minute}, {60, time.Minute},
	} {
		if got := r.Teto(c.tentativa); got != c.teto {
			t.Errorf("tentativa %d: teto %v; esperado %v",
				c.tentativa, got, c.teto)
		}
		metade := r.Delay(c.tentativa, func() float64 { return 0.5 })
		if metade != c.teto/2 {
			t.Errorf("tentativa %d: com 0,5, %v", c.tentativa, metade)
		}
		if zero := r.Delay(c.tentativa, func() float64 { return 0 }); zero != 0 {
			t.Errorf("tentativa %d: com 0, %v", c.tentativa, zero)
		}
	}
}
