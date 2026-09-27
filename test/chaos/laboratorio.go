//go:build chaos

package chaos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/test/testutil"
)

// Binarios são o enxamed e o exemplo do cluster do Capítulo 26
// (worker, carga, conferência), compilados uma vez por execução.
type Binarios struct{ Enxamed, Cluster string }

// Compilar compila os binários em dir.
func Compilar(ctx context.Context, dir string) (Binarios, error) {
	b := Binarios{Enxamed: filepath.Join(dir, "enxamed"),
		Cluster: filepath.Join(dir, "cluster")}
	for alvo, pkg := range map[string]string{
		b.Enxamed: "../../cmd/enxamed",
		b.Cluster: "../../examples/cap26/cluster"} {
		out, err := exec.CommandContext(ctx, "go", "build", "-o", alvo,
			pkg).
			CombinedOutput()
		if err != nil {
			return b, fmt.Errorf("%s: %w\n%s", pkg, err, out)
		}
	}
	return b, nil
}

// No é um enxamed do laboratório, com o proxy dele até o banco.
type No struct {
	Nome, HTTP, GRPC string
	Proxy            *Proxy
	Log              string // arquivo com a saída do processo
	cmd              *exec.Cmd
	worker           *exec.Cmd
}

// Laboratorio é um cluster de três enxamed, cada um com um worker
// remoto, num banco novo.
type Laboratorio struct {
	T   *testing.T
	Bin Binarios
	DSN string
	DB  *pgxpool.Pool
	Nos []*No
	dir string
}

// livro:inicio laboratorio

// NovoLaboratorio sobe três enxamed num banco novo, cada um falando com
// o PostgreSQL pelo próprio proxy, espera as 512 partições terem dono
// e conecta um worker remoto a cada nó. env acrescenta variáveis de
// ambiente ao processo do nó i (de 1 a 3).
func NovoLaboratorio(t *testing.T, bin Binarios,
	env func(i int) []string) *Laboratorio {
	l := &Laboratorio{T: t, Bin: bin, DSN: testutil.PostgresDSN(t),
		dir: t.TempDir()}
	db, err := pgxpool.New(t.Context(), l.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	l.DB = db
	u, err := url.Parse(l.DSN)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		n := &No{Nome: fmt.Sprintf("no-%d", i),
			HTTP:  fmt.Sprintf("127.0.0.1:1918%d", i),
			GRPC:  fmt.Sprintf("127.0.0.1:1928%d", i),
			Proxy: &Proxy{Destino: u.Host}}
		addr, err := n.Proxy.Abrir()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(n.Proxy.Fechar)
		viaProxy := *u
		viaProxy.Host = addr
		args := []string{"-dsn", viaProxy.String(), "-no", n.Nome,
			"-http", n.HTTP, "-grpc", n.GRPC, "-tokens", "t1:loja",
			"-worker-token", "w", "-aviso", "0", "-resgate", "5s"}
		n.Log = filepath.Join(l.dir, n.Nome+".log")
		var vars []string
		if env != nil {
			vars = env(i)
		}
		n.cmd = l.processo(n.Log, vars, bin.Enxamed, args...)
		l.Nos = append(l.Nos, n)
	}
	l.Esperar(30*time.Second, "as 512 partições com dono", func() bool {
		return l.Contar(`SELECT count(*) FROM partition_lease
			WHERE owner IS NOT NULL
			  AND lease_expires_at > now()`) == 512
	})
	l.rodar(bin.Cluster, "preparar", "-dsn", l.DSN)
	for _, n := range l.Nos {
		n.worker = l.processo(filepath.Join(l.dir, "w-"+n.Nome+".log"),
			nil, bin.Cluster, "worker", "-dsn", l.DSN, "-grpc", n.GRPC,
			"-nome", "w-"+n.Nome)
	}
	return l
}

// livro:fim laboratorio

func (l *Laboratorio) processo(log string, env []string, bin string,
	args ...string) *exec.Cmd {
	f, err := os.Create(log)
	if err != nil {
		l.T.Fatal(err)
	}
	// O processo vive até a limpeza do teste, não até o fim de um ctx.
	c := exec.Command(bin, args...) //nolint:noctx
	c.Stdout, c.Stderr = f, f
	c.Env = append(os.Environ(), env...)
	if err := c.Start(); err != nil {
		l.T.Fatal(err)
	}
	l.T.Cleanup(func() {
		_ = c.Process.Kill()
		_ = c.Wait()
		_ = f.Close()
	})
	return c
}

func (l *Laboratorio) rodar(bin string, args ...string) string {
	out, err := exec.CommandContext(l.T.Context(), bin, args...).
		CombinedOutput()
	if err != nil {
		l.T.Fatalf("%s %v: %v\n%s", bin, args, err, out)
	}
	return string(out)
}

// Contar roda uma consulta que devolve um número.
func (l *Laboratorio) Contar(sql string) int {
	return l.contar(l.T.Context(), sql)
}

func (l *Laboratorio) contar(ctx context.Context, sql string) int {
	var n int
	_ = l.DB.QueryRow(ctx, sql).Scan(&n)
	return n
}

// Aguardar deixa o experimento correr por d, em tempo real: a falha
// dura o que dura no mundo.
func (l *Laboratorio) Aguardar(d time.Duration) {
	select {
	case <-time.After(d):
	case <-l.T.Context().Done():
	}
}

// Esperar espera cond valer, ou falha depois do prazo.
func (l *Laboratorio) Esperar(prazo time.Duration, o string,
	cond func() bool) time.Duration {
	inicio := time.Now()
	for !cond() {
		if time.Since(inicio) > prazo {
			l.despejar()
			l.T.Fatalf("%s: não aconteceu em %v", o, prazo)
		}
		l.Aguardar(100 * time.Millisecond)
	}
	return time.Since(inicio)
}

// Lider devolve o nó que tem a liderança do pgcoord.
func (l *Laboratorio) Lider() *No {
	var nome string
	_ = l.DB.QueryRow(l.T.Context(), `SELECT node
		FROM coord_leader`).Scan(&nome)
	for _, n := range l.Nos {
		if n.Nome == nome {
			return n
		}
	}
	return nil
}

// Outro devolve um nó diferente dos dados.
func (l *Laboratorio) Outro(exceto ...*No) *No {
	for _, n := range l.Nos {
		if !slices.Contains(exceto, n) {
			return n
		}
	}
	return nil
}

// Carga enfileira n jobs pela API do nó, à taxa dada, e devolve quem
// espera o fim dela.
func (l *Laboratorio) Carga(n *No, jobs, taxa int) func() error {
	var erros strings.Builder
	c := exec.CommandContext(l.T.Context(), l.Bin.Cluster, "carga",
		"-api", "http://"+n.HTTP,
		"-n", fmt.Sprint(jobs), "-taxa", fmt.Sprint(taxa))
	c.Stderr = &erros
	if err := c.Start(); err != nil {
		l.T.Fatal(err)
	}
	return func() error {
		if err := c.Wait(); err != nil {
			return fmt.Errorf("%w: %s", err, erros.String())
		}
		return nil
	}
}

// Terminados conta os jobs da carga concluídos.
func (l *Laboratorio) Terminados() int {
	return l.terminados(l.T.Context())
}

func (l *Laboratorio) terminados(ctx context.Context) int {
	return l.contar(ctx, `SELECT count(*) FROM job
		WHERE queue = 'carga' AND state = 'completed'`)
}

// Conferir é a conferência do Capítulo 26: perdidos, repetidos, fora
// de ordem.
func (l *Laboratorio) Conferir(jobs int) string {
	return l.rodar(l.Bin.Cluster, "conferir", "-dsn", l.DSN, "-n",
		fmt.Sprint(jobs))
}

// Posses conta as partições com posse válida de cada nó.
func (l *Laboratorio) Posses() map[string]int {
	return l.posses(l.T.Context())
}

func (l *Laboratorio) posses(ctx context.Context) map[string]int {
	rows, err := l.DB.Query(ctx, `SELECT owner,
		count(*) FROM partition_lease WHERE owner IS NOT NULL
		  AND lease_expires_at > now() GROUP BY owner`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var o string
		var c int
		if rows.Scan(&o, &c) == nil {
			m[o] = c
		}
	}
	return m
}

// livro:inicio amostrar

// Amostrar anota, a cada intervalo e até ctx terminar, os jobs
// concluídos e as partições de cada nó: o steady state do experimento,
// antes, durante e depois da falha. A série sai de fim, depois de ctx.
func (l *Laboratorio) Amostrar(ctx context.Context,
	intervalo time.Duration) (fim func() []Amostra) {
	var as []Amostra
	pronto := make(chan struct{})
	inicio := time.Now()
	go func() {
		defer close(pronto)
		for ctx.Err() == nil {
			as = append(as, Amostra{T: time.Since(inicio),
				Concluidos: l.terminados(ctx), Posses: l.posses(ctx)})
			select {
			case <-ctx.Done():
			case <-time.After(intervalo):
			}
		}
	}()
	return func() []Amostra { <-pronto; return as }
}

// livro:fim amostrar

// Amostra é um ponto da série.
type Amostra struct {
	T          time.Duration
	Concluidos int
	Posses     map[string]int
}

func escrever(t *testing.T, arq, conteudo string) {
	_ = os.MkdirAll(filepath.Dir(arq), 0o755)
	if err := os.WriteFile(arq, []byte(conteudo), 0o644); err != nil {
		t.Error(err)
	}
}

// despejar mostra os jobs da carga que não terminaram e o fim do log
// de cada nó, para o diagnóstico.
func (l *Laboratorio) despejar() {
	rows, err := l.DB.Query(l.T.Context(), `SELECT j.state,
		j.partition_id, j.attempt, coalesce(j.attempted_by, ''),
		coalesce(p.owner, ''), p.range_id, count(*)
		FROM job j JOIN partition_lease p USING (partition_id)
		WHERE j.queue = 'carga' AND j.state <> 'completed'
		GROUP BY 1, 2, 3, 4, 5, 6 ORDER BY 2 LIMIT 30`)
	if err == nil {
		for rows.Next() {
			var st, por, dono string
			var part, tent, n int
			var rid int64
			_ = rows.Scan(&st, &part, &tent, &por, &dono, &rid, &n)
			l.T.Logf("pendente: %d job(s) %s, partição %d (dono %s, "+
				"token %d), tentativa %d por %q", n, st, part, dono,
				rid, tent, por)
		}
		rows.Close()
	}
	for _, n := range l.Nos {
		w, _ := os.ReadFile(filepath.Join(l.dir, "w-"+n.Nome+".log"))
		if len(w) > 0 {
			l.T.Logf("== worker de %s\n%s", n.Nome, w)
		}
		b, _ := os.ReadFile(n.Log)
		linhas := strings.Split(strings.TrimSpace(string(b)), "\n")
		if len(linhas) > 25 {
			linhas = linhas[len(linhas)-25:]
		}
		l.T.Logf("== %s (posses %v)\n%s", n.Nome, l.Posses(),
			strings.Join(linhas, "\n"))
	}
}

// Visao é o que o nó de acha do nó sobre, pela API: estado, phi e
// silêncio em milissegundos.
func (l *Laboratorio) Visao(de, sobre *No) (string, float64, int64) {
	req, err := http.NewRequestWithContext(l.T.Context(),
		http.MethodGet,
		"http://"+de.HTTP+"/v1/cluster/members", nil)
	if err != nil {
		return "", 0, 0
	}
	req.Header.Set("Authorization", "Bearer t1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, 0
	}
	defer func() { _ = resp.Body.Close() }()
	var lista struct {
		Items []struct {
			Node    string  `json:"node"`
			State   string  `json:"state"`
			Phi     float64 `json:"phi"`
			Silence int64   `json:"silence_ms"`
		} `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&lista)
	for _, m := range lista.Items {
		if m.Node == sobre.Nome {
			return m.State, m.Phi, m.Silence
		}
	}
	return "", 0, 0
}

// NoMapa diz se o nó está no último mapa do coordenador, e a época.
func (l *Laboratorio) NoMapa(n *No) (bool, int) {
	var epoca int
	var tem bool
	_ = l.DB.QueryRow(l.T.Context(), `SELECT epoch,
		owners ? $1 FROM coord_assignment
		ORDER BY epoch DESC LIMIT 1`, n.Nome).Scan(&epoca, &tem)
	return tem, epoca
}

// ConcluidosPor conta os jobs da carga concluídos pelo worker do nó.
func (l *Laboratorio) ConcluidosPor(n *No) int {
	var c int
	_ = l.DB.QueryRow(l.T.Context(), `SELECT count(*) FROM job
		WHERE queue = 'carga' AND state = 'completed'
		  AND attempted_by = $1`, "w-"+n.Nome).Scan(&c)
	return c
}

// GuardarLogs copia os logs dos nós e dos workers para testdata/, onde
// sobrevivem ao teste.
func (l *Laboratorio) GuardarLogs(nome string) {
	for _, n := range l.Nos {
		for _, arq := range []string{n.Nome + ".log",
			"w-" + n.Nome + ".log"} {
			b, _ := os.ReadFile(filepath.Join(l.dir, arq))
			escrever(l.T, filepath.Join("testdata", nome, arq),
				string(b))
		}
	}
}

// Tomadas conta as partições que o último mapa dá ao nó e que estão,
// agora, com a posse válida de outro nó.
func (l *Laboratorio) Tomadas(n *No) int {
	var c int
	_ = l.DB.QueryRow(l.T.Context(), `SELECT count(*)
		FROM partition_lease p, (SELECT owners FROM coord_assignment
		  ORDER BY epoch DESC LIMIT 1) a
		WHERE a.owners->>p.partition_id = $1 AND p.owner <> $1
		  AND p.lease_expires_at > now()`, n.Nome).Scan(&c)
	return c
}

// RenovacoesEsperando conta as renovações de posse (de todos os nós)
// esperando um lock no banco, agora.
func (l *Laboratorio) RenovacoesEsperando() int {
	return l.Contar(`SELECT count(*) FROM pg_stat_activity
		WHERE wait_event_type = 'Lock'
		  AND query LIKE '%SET lease_expires_at = now()%'`)
}
