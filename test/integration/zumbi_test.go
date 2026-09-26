//go:build integration

package integration_test

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/test/testutil"
)

// processo sobe um nó do exemplo zumbi e entrega as linhas que ele
// imprime num canal.
func processo(t *testing.T, bin string, args ...string) (*exec.Cmd,
	<-chan string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...)
	saida, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	linhas := make(chan string, 100)
	go func() {
		defer close(linhas)
		s := bufio.NewScanner(saida)
		for s.Scan() {
			linhas <- s.Text()
		}
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGCONT)
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	})
	return cmd, linhas
}

// livro:inicio zumbi-teste

// O dono A é pausado com SIGSTOP depois de escrever; B assume quando o
// lease de A vence; A volta com SIGCONT. Sem cerca, A escreve com a
// numeração que tinha na memória, e o extrato ganha números repetidos.
// Com cerca, a primeira escrita de A depois da pausa é recusada.
func TestZumbi(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "zumbi")
	rodar(t, "go", "build", "-o", bin, "../../examples/cap24/zumbi")
	for _, modo := range []string{"ingenuo", "cerca"} {
		t.Run(modo, func(t *testing.T) {
			dsn := testutil.PostgresDSN(t)
			db, err := pgxpool.New(t.Context(), dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			rodar(t, bin, "-dsn", dsn, "preparar")
			args := func(no string) []string {
				a := []string{"-dsn", dsn, "-no", no}
				if modo == "cerca" {
					a = append(a, "-cerca")
				}
				return a
			}
			a, linhasA := processo(t, bin, args("a")...)
			escritas := func(no string) int {
				return contar(t, db, `SELECT count(*) FROM extrato
					WHERE no = '`+no+`'`)
			}
			esperar(t, func() bool { return escritas("a") >= 5 })
			_ = a.Process.Signal(syscall.SIGSTOP)
			processo(t, bin, args("b")...)
			esperar(t, func() bool { return escritas("b") >= 10 })
			_ = a.Process.Signal(syscall.SIGCONT)
			fim := esperarLinha(t, linhasA, "perdi a posse",
				"escrita recusada")
			t.Logf("A depois da pausa: %s", fim)
			t.Log("\n" + rodar(t, bin, "-dsn", dsn, "conferir"))
			dup := contar(t, db, `SELECT count(*) FROM (SELECT seq
				FROM extrato GROUP BY seq HAVING count(*) > 1) d`)
			switch {
			case modo == "ingenuo" && dup == 0:
				t.Fatal("o zumbi não escreveu por cima de B")
			case modo == "cerca" && dup > 0:
				t.Fatalf("%d números repetidos com a cerca", dup)
			case modo == "cerca" &&
				!strings.Contains(fim, "escrita recusada"):
				t.Fatalf("A não foi barrado: %s", fim)
			}
		})
	}
}

// livro:fim zumbi-teste

// esperarLinha espera o processo imprimir uma linha com um dos textos.
func esperarLinha(t *testing.T, linhas <-chan string,
	textos ...string) string {
	t.Helper()
	limite := time.After(15 * time.Second)
	for {
		select {
		case l, ok := <-linhas:
			if !ok {
				t.Fatal("o processo terminou sem avisar")
			}
			for _, x := range textos {
				if strings.Contains(l, x) {
					return l
				}
			}
		case <-limite:
			t.Fatalf("nenhuma linha com %v em 15 s", textos)
		}
	}
}
