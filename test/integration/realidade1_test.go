//go:build integration

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// livro:inicio realidade1

// Teste de Realidade #1: 500 jobs e 50 workflows; o worker leva kill -9
// no meio; outro processo termina o trabalho. Nenhum job perdido,
// nenhum workflow perdido, nenhum efeito duplicado.
func TestRealidade1(t *testing.T) {
	if os.Getenv("ENXAME_DB_DSN") == "" {
		t.Skip("ENXAME_DB_DSN não definido: rode make up")
	}
	bin := filepath.Join(t.TempDir(), "realidade")
	rodar(t, "go", "build", "-o", bin, "../../examples/cap17/realidade")
	rodar(t, bin, "preparar")

	primeiro := exec.Command(bin, "trabalhar")
	if err := primeiro.Start(); err != nil {
		t.Fatal(err)
	}
	db := conectar(t)
	for contar(t, db, `SELECT count(*) FROM efeito`) < 200 {
		<-time.After(20 * time.Millisecond)
	}
	if err := primeiro.Process.Kill(); err != nil { // SIGKILL
		t.Fatal(err)
	}
	_ = primeiro.Wait()
	t.Logf("kill -9 com %d efeitos gravados",
		contar(t, db, `SELECT count(*) FROM efeito`))

	rodar(t, bin, "trabalhar") // termina quando não sobra trabalho
	saida := rodar(t, bin, "conferir")
	t.Log("\n" + saida)
	esperado := map[string]int{
		"jobs de tarefa concluídos":  500,
		"jobs não concluídos":        0,
		"workflows concluídos":       50,
		"efeitos (chaves distintas)": 600, // 500 jobs + 50 × 2 passos
	}
	for linha := range strings.Lines(saida) {
		campos := strings.Fields(linha)
		n, _ := strconv.Atoi(campos[len(campos)-1])
		nome := strings.Join(campos[:len(campos)-1], " ")
		if e, ok := esperado[nome]; ok && n != e {
			t.Errorf("%s: %d; esperado %d", nome, n, e)
		}
	}
}

// livro:fim realidade1

func rodar(t *testing.T, nome string, args ...string) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), nome,
		args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", nome, args, err, out)
	}
	return string(out)
}

func conectar(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(os.Getenv("ENXAME_DB_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = "enxame_realidade1"
	db, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}
