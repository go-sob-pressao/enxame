//go:build chaos

package chaos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// livro:inicio experimento-disco

// Experimento 4 — o disco de dados do banco cheio.
//
// Hipótese: com o disco das tabelas cheio, a API recusa os jobs novos
// com um erro, em vez de aceitá-los e perdê-los; todo job aceito (201)
// roda; e, quando o espaço volta, o cluster volta a aceitar e a
// trabalhar em menos de 10 s, sem reiniciar nenhum processo.
// Injeção: um PostgreSQL à parte, com as tabelas num tablespace de 64
// MiB em memória (o WAL fica fora dele); uma tabela de lastro enche o
// tablespace com a carga rodando; 20 s depois, o lastro é apagado.
// Raio de alcance: um contêiner criado e destruído pelo teste.
// Refuta: um job aceito que não rodou; um 201 para um job que não foi
// gravado; um processo do cluster que terminou; mais de 10 s para
// voltar a aceitar depois do espaço liberado.
func TestDiscoCheio(t *testing.T) {
	dsn := bancoComDiscoPequeno(t)
	t.Setenv("ENXAME_DB_DSN", dsn)
	l := NovoLaboratorio(t, bin, nil)
	entrada := l.Outro(l.Lider())
	ctx, parar := context.WithCancel(t.Context())
	defer parar()
	serie := l.Amostrar(ctx, 500*time.Millisecond)
	fimDaCarga := cargaTolerante(ctx, entrada, 50)

	l.Aguardar(3 * time.Second)
	cheio := encherDisco(t, l.DSN)
	t.Logf("disco cheio %v depois do começo do lastro",
		cheio.Round(100*time.Millisecond))
	l.Aguardar(20 * time.Second)
	if _, err := l.DB.Exec(t.Context(), `DROP TABLE lastro`); err != nil {
		t.Fatalf("liberar o espaço: %v", err)
	}
	liberado := time.Now()
	l.Aguardar(15 * time.Second)
	parar()
	r := fimDaCarga()
	volta := r.primeiroAceitoDepois(liberado)
	l.Esperar(2*time.Minute, "os jobs aceitos concluídos", func() bool {
		return l.Terminados() >= len(r.aceitos)
	})
	var rodaram int
	_ = l.DB.QueryRow(t.Context(), `SELECT count(*) FROM efeito
		WHERE n = ANY($1)`, r.aceitos).Scan(&rodaram)
	vivos := 0
	for _, n := range l.Nos {
		if l.Vivo(n) {
			vivos++
		}
	}
	t.Logf("aceitos %d, recusados %v; aceitos que rodaram %d; "+
		"primeiro aceito %v depois do espaço liberado; processos "+
		"vivos %d de 3", len(r.aceitos), r.recusas, rodaram,
		volta.Round(100*time.Millisecond), vivos)
	registrarSerie(t, "disco", serie())
	l.GuardarLogs("disco")
	if rodaram != len(r.aceitos) || vivos != 3 || volta > 10*time.Second ||
		volta < 0 {
		t.Fatal("hipótese refutada")
	}
}

// livro:fim experimento-disco

// bancoComDiscoPequeno sobe um PostgreSQL com um tablespace de 64 MiB
// em tmpfs como padrão para as tabelas, e devolve a DSN de admin.
func bancoComDiscoPequeno(t *testing.T) string {
	const nome, porta = "enxame-caos-disco", "55433"
	docker := func(args ...string) string {
		out, err := exec.CommandContext(t.Context(), "docker", args...).
			CombinedOutput()
		if err != nil {
			t.Fatalf("docker %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	_ = exec.CommandContext(t.Context(), "docker", "rm", "-f", nome).
		Run()
	docker("run", "-d", "--rm", "--name", nome,
		"-e", "POSTGRES_PASSWORD=enxame", "-p", porta+":5432",
		"--mount", "type=tmpfs,destination=/disco,tmpfs-size=67108864",
		"postgres:18.6")
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", nome).Run() //nolint:noctx
	})
	dsn := "postgres://postgres:enxame@localhost:" + porta +
		"/postgres?sslmode=disable"
	var conn *pgx.Conn
	for range 60 {
		var err error
		if conn, err = pgx.Connect(t.Context(), dsn); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond) //nolint:forbidigo // subida
	}
	if conn == nil {
		t.Fatal("o banco do experimento não subiu")
	}
	defer func() { _ = conn.Close(t.Context()) }()
	docker("exec", "-u", "root", nome, "chown", "postgres", "/disco")
	for _, sql := range []string{
		`CREATE TABLESPACE pequeno LOCATION '/disco'`,
		`ALTER SYSTEM SET default_tablespace = 'pequeno'`,
		`SELECT pg_reload_conf()`} {
		if _, err := conn.Exec(t.Context(), sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	return dsn
}

// encherDisco cria o lastro e o faz crescer até o tablespace acabar;
// devolve quanto tempo levou.
func encherDisco(t *testing.T, dsn string) time.Duration {
	inicio := time.Now()
	conn, err := pgx.Connect(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(t.Context()) }()
	if _, err := conn.Exec(t.Context(),
		`CREATE TABLE lastro (x text)`); err != nil {
		t.Fatal(err)
	}
	for {
		_, err := conn.Exec(t.Context(), `INSERT INTO lastro
			SELECT string_agg(md5(random()::text), '')
			FROM generate_series(1, 64), generate_series(1, 200) g
			GROUP BY g`)
		if err != nil {
			if strings.Contains(err.Error(), "No space left") ||
				strings.Contains(err.Error(), "could not extend") {
				return time.Since(inicio)
			}
			t.Fatalf("lastro: %v", err)
		}
	}
}

// resultadoDaCarga é o que a carga tolerante viu.
type resultadoDaCarga struct {
	aceitos   []int
	quando    []time.Time // instante de cada aceite
	recusas   map[int]int // status → quantas
	mu        sync.Mutex
	encerrada chan struct{}
}

func (r *resultadoDaCarga) primeiroAceitoDepois(t time.Time) time.Duration {
	for _, q := range r.quando {
		if q.After(t) {
			return q.Sub(t)
		}
	}
	return -1
}

// cargaTolerante enfileira jobs à taxa dada até ctx terminar, como a
// carga do Cap. 26, mas sem parar no primeiro erro: conta os aceitos e
// as recusas por status.
func cargaTolerante(ctx context.Context, n *No,
	taxa int) func() *resultadoDaCarga {
	r := &resultadoDaCarga{recusas: map[int]int{},
		encerrada: make(chan struct{})}
	go func() {
		defer close(r.encerrada)
		seq := make([]int, 20)
		t := time.NewTicker(time.Second / time.Duration(taxa))
		defer t.Stop()
		for i := 0; ctx.Err() == nil; i++ {
			k := i % 20
			corpo, _ := json.Marshal(map[string]any{
				"queue": "carga", "kind": "trabalho",
				"ordering_key": fmt.Sprintf("chave-%d", k),
				"args": map[string]int{"n": i, "chave": k,
					"seq": seq[k]}})
			seq[k]++
			status := postar(ctx, "http://"+n.HTTP+"/v1/jobs", corpo)
			r.mu.Lock()
			if status == http.StatusCreated {
				r.aceitos = append(r.aceitos, i)
				r.quando = append(r.quando, time.Now())
			} else if ctx.Err() == nil {
				r.recusas[status]++
			}
			r.mu.Unlock()
			select {
			case <-t.C:
			case <-ctx.Done():
			}
		}
	}()
	return func() *resultadoDaCarga { <-r.encerrada; return r }
}

func postar(ctx context.Context, url string, corpo []byte) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		bytes.NewReader(corpo))
	if err != nil {
		return 0
	}
	req.Header.Set("Authorization", "Bearer t1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}
