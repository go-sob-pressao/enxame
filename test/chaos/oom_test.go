//go:build chaos

package chaos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio experimento-oom

// Experimento 5 — memória no limite.
//
// Hipótese: num contêiner de 128 MiB, um enxamed com os limites padrão
// — corpo de 1 MiB, 32 requisições em curso — aguenta 64 clientes
// mandando jobs de quase 1 MiB sem ser morto pelo OOM killer.
// Refutada na primeira execução (Cap. 28); o experimento ficou com a
// medida que a substituiu: com 256 MiB e GOMEMLIMIT=200MiB, aguenta.
// Injeção: o enxamed num contêiner Linux com --memory; 64 clientes que
// não respeitam o Retry-After, 20 s de rajada.
// Raio de alcance: um contêiner criado pelo teste, com banco efêmero.
// Refuta: o contêiner de 256 MiB morto por falta de memória.
func TestMemoriaNoLimite(t *testing.T) {
	for _, c := range []struct {
		memoria, gomemlimit string
		deveViver           bool
	}{
		{"128m", "", false}, // a referência: morre
		{"128m", "100MiB", false},
		{"256m", "200MiB", true},
	} {
		nome := c.memoria + " GOMEMLIMIT=" + c.gomemlimit
		t.Run(nome, func(t *testing.T) {
			api, fim := noEmConteiner(t, c.memoria, c.gomemlimit)
			status := rajada(t, api, 64, 20*time.Second, 1000*1000)
			oom, pico := fim()
			t.Logf("%s: 201 %d, 503 %d; OOMKilled %v; pico %d MiB",
				nome, status[201], status[503], oom, pico>>20)
			if c.deveViver && oom {
				t.Error("hipótese refutada: morto por falta de memória")
			}
		})
	}
}

// livro:fim experimento-oom

// noEmConteiner sobe um enxamed num contêiner com o limite de memória
// dado e devolve o endereço da API e quem encerra o contêiner, dizendo
// se ele morreu por falta de memória e qual foi o pico.
func noEmConteiner(t *testing.T, memoria, gomemlimit string,
	extra ...string) (string, func() (bool, int64)) {
	dir := t.TempDir()
	alvo := filepath.Join(dir, "enxamed")
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", alvo,
		"../../cmd/enxamed")
	cmd.Env = append(cmd.Environ(), "GOOS=linux", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	u, err := url.Parse(testutil.PostgresDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	u.Host = "host.docker.internal:" + u.Port()
	nome := "enxame-caos-oom"
	_ = exec.CommandContext(t.Context(), "docker", "rm", "-f", nome).
		Run()
	args := []string{"run", "-d", "--name", nome,
		"--memory", memoria, "--memory-swap", memoria,
		"-p", "19190:8080", "-v", dir + ":/caos:ro",
		"--entrypoint", "/caos/enxamed"}
	if gomemlimit != "" {
		args = append(args, "-e", "GOMEMLIMIT="+gomemlimit)
	}
	args = append(args, "postgres:18.6", "-dsn", u.String(),
		"-http", ":8080", "-grpc", ":7233", "-tokens", "t1:loja",
		"-worker-token", "w", "-no", "no-oom")
	args = append(args, extra...)
	if out, err := exec.CommandContext(t.Context(), "docker",
		args...).CombinedOutput(); err != nil {
		t.Fatalf("docker run: %v\n%s", err, out)
	}
	api := "http://127.0.0.1:19190"
	for range 60 {
		if resp, err := http.Get(api + "/healthz"); err == nil { //nolint:noctx
			_ = resp.Body.Close()
			break
		}
		time.Sleep(500 * time.Millisecond) //nolint:forbidigo // subida
	}
	return api, func() (bool, int64) {
		out, _ := exec.CommandContext(context.Background(), "docker",
			"inspect", "-f", "{{.State.OOMKilled}}", nome).Output()
		var pico int64
		if p, err := exec.CommandContext(context.Background(), "docker",
			"exec", nome, "cat", "/sys/fs/cgroup/memory.peak").
			Output(); err == nil {
			_, _ = fmt.Sscan(string(p), &pico)
		}
		logs, _ := exec.CommandContext(context.Background(), "docker",
			"logs", "--tail", "5", nome).CombinedOutput()
		t.Logf("fim do log:\n%s", logs)
		_ = exec.CommandContext(context.Background(), "docker", "rm",
			"-f", nome).Run()
		return strings.TrimSpace(string(out)) == "true", pico
	}
}

// rajada põe n clientes mandando jobs de tamanho bytes durante d, e
// conta as respostas por status (0: sem resposta).
func rajada(t *testing.T, api string, n int, d time.Duration,
	bytes int) map[int]int {
	corpo, _ := json.Marshal(map[string]any{"queue": "grande",
		"kind": "grande", "args": map[string]string{
			"dados": strings.Repeat("x", bytes-200)}})
	ctx, parar := context.WithTimeout(t.Context(), d)
	defer parar()
	var mu sync.Mutex
	status := map[int]int{}
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			for ctx.Err() == nil {
				s := postar(ctx, api+"/v1/jobs", corpo)
				if ctx.Err() != nil {
					return
				}
				mu.Lock()
				status[s]++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return status
}
