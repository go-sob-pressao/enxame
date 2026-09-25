package job_test

import (
	"flag"
	"os"
	"testing"
)

// TestMain aumenta o número de casos do rapid de 100 para 5.000, a
// menos que -rapid.checks venha na linha de comando. Com 100, a
// propriedade da máquina de estados deixava passar defeitos que só
// aparecem depois de algumas centenas de sequências.
func TestMain(m *testing.M) {
	flag.Parse()
	informado := false
	flag.Visit(func(f *flag.Flag) {
		informado = informado || f.Name == "rapid.checks"
	})
	if !informado {
		_ = flag.Set("rapid.checks", "5000")
	}
	os.Exit(m.Run())
}
