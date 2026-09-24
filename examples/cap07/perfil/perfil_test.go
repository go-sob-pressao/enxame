package main

import (
	"bytes"
	"runtime/pprof"
	"strings"
	"testing"
)

// O perfil goroutineleak vê o vazamento local e não vê o da variável
// global.
func TestPerfilGoroutineleak(t *testing.T) {
	for range 10 {
		consultar()
		vazarEmGlobal()
	}
	var buf bytes.Buffer
	if err := pprof.Lookup("goroutineleak").WriteTo(&buf, 1); err != nil {
		t.Fatal(err)
	}
	perfil := buf.String()
	if !strings.Contains(perfil, "consultar.func1") {
		t.Errorf("o perfil não mostrou o vazamento local:\n%s", perfil)
	}
	if strings.Contains(perfil, "vazarEmGlobal.func1") {
		t.Errorf(
			"o perfil mostrou a goroutine presa na variável global",
		)
	}
	t.Logf("%.400s", perfil)
}
