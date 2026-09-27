//go:build integration

package integration_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio realidade3

// Teste de Realidade #3: três enxamed, um worker remoto em cada, 600
// jobs com chave de ordem chegando a 100 por segundo. Três segundos
// depois do começo da carga, o líder e o worker dele levam kill -9.
// Todo job roda, nenhum efeito se repete, e a ordem de cada chave vale.
func TestRealidade3(t *testing.T) {
	if comRace {
		t.Skip("teste de processos com carga: rode sem -race")
	}
	dir := t.TempDir()
	enxamed := filepath.Join(dir, "enxamed")
	cluster := filepath.Join(dir, "cluster")
	rodar(t, "go", "build", "-o", enxamed, "../../cmd/enxamed")
	rodar(t, "go", "build", "-o", cluster, "../../examples/cap26/cluster")
	dsn := testutil.PostgresDSN(t)
	db, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	nos, workers := map[string]*exec.Cmd{}, map[string]*exec.Cmd{}
	for i := 1; i <= 3; i++ {
		no := fmt.Sprintf("no-%d", i)
		nos[no], _ = processo(t, enxamed, "-dsn", dsn, "-no", no,
			"-http", fmt.Sprintf(":1818%d", i),
			"-grpc", fmt.Sprintf(":1828%d", i), "-tokens", "t1:loja",
			"-worker-token", "w", "-aviso", "0", "-resgate", "5s")
	}
	esperarPor(t, 30*time.Second, func() bool { // as 512 com dono
		return contar(t, db, `SELECT count(*) FROM partition_lease
			WHERE owner IS NOT NULL
			  AND lease_expires_at > now()`) == 512
	})
	rodar(t, cluster, "preparar", "-dsn", dsn)
	for i := 1; i <= 3; i++ {
		workers[fmt.Sprintf("no-%d", i)], _ = processo(t, cluster,
			"worker", "-dsn", dsn, "-grpc",
			fmt.Sprintf("127.0.0.1:1828%d", i), "-nome",
			fmt.Sprintf("w%d", i))
	}
	// A carga entra por um nó que não é o líder: o líder vai cair.
	var lider string
	_ = db.QueryRow(t.Context(), `SELECT node FROM coord_leader`).
		Scan(&lider)
	porta := "18181"
	if lider == "no-1" {
		porta = "18182"
	}
	var erros strings.Builder
	carga := exec.CommandContext(t.Context(), cluster, "carga",
		"-api", "http://127.0.0.1:"+porta, "-n", "600", "-taxa", "100")
	carga.Stderr = &erros
	if err := carga.Start(); err != nil {
		t.Fatal(err)
	}
	<-time.After(3 * time.Second)
	_ = nos[lider].Process.Signal(syscall.SIGKILL)
	_ = workers[lider].Process.Signal(syscall.SIGKILL)
	morte := time.Now()
	if err := carga.Wait(); err != nil {
		t.Fatalf("carga: %v: %s", err, erros.String())
	}
	var donos time.Duration
	esperarPor(t, 90*time.Second, func() bool {
		if donos == 0 && contar(t, db, `SELECT count(*)
			FROM partition_lease WHERE owner <> '`+lider+`'
			  AND lease_expires_at > now()`) == 512 {
			donos = time.Since(morte)
		}
		return contar(t, db, `SELECT count(*) FROM job
			WHERE queue = 'carga' AND state <> 'completed'`) == 0 &&
			contar(t, db, `SELECT count(*) FROM job
				WHERE queue = 'carga'`) == 600
	})
	t.Logf("kill -9 no líder %s; partições com dono de novo em %v; "+
		"último job %v depois do kill", lider,
		donos.Round(100*time.Millisecond),
		time.Since(morte).Round(100*time.Millisecond))
	saida := rodar(t, cluster, "conferir", "-dsn", dsn, "-n", "600")
	t.Log("\n" + saida)
	if !strings.Contains(saida, "perdidos: 0;") ||
		!strings.Contains(saida, "fora de ordem: 0") {
		t.Fatal("o cluster perdeu jobs ou a ordem")
	}
}

// livro:fim realidade3

func esperarPor(t *testing.T, prazo time.Duration, cond func() bool) {
	t.Helper()
	limite := time.After(prazo)
	for !cond() {
		select {
		case <-limite:
			t.Fatalf("condição não chegou em %v", prazo)
		case <-time.After(200 * time.Millisecond):
		}
	}
}
