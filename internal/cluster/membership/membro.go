package membership

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
)

// Config configura o membership de um nó.
type Config struct {
	No        coordinator.NodeID
	Endereco  string
	DB        *pgxpool.Pool
	Intervalo time.Duration   // entre batidas (500 ms)
	Detector  func() Detector // um por nó observado (NovoPhi)
	Log       *slog.Logger
}

// Situacao é o que este nó acha de outro, agora.
type Situacao struct {
	No       coordinator.NodeID
	Endereco string
	Estado   Estado
	Phi      float64       // só com o detector Phi
	Silencio time.Duration // desde a última batida, no relógio do banco
}

type observado struct {
	detector Detector
	inicio   time.Time // started_at: um reinício zera a história
	ultima   time.Time // beat_at da última batida vista
	endereco string
}

// Membro bate pelo nó e observa os outros.
type Membro struct {
	cfg Config

	mu     sync.Mutex
	outros map[coordinator.NodeID]*observado
	agora  time.Time // now() do banco na última leitura
	viu    bool      // já leu a tabela inteira ao menos uma vez
}

// Novo cria o membership do nó; Run o põe para bater.
func Novo(cfg Config) *Membro {
	if cfg.Intervalo <= 0 {
		cfg.Intervalo = 500 * time.Millisecond
	}
	if cfg.Detector == nil {
		cfg.Detector = func() Detector { return NovoPhi(cfg.Intervalo) }
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	return &Membro{cfg: cfg,
		outros: map[coordinator.NodeID]*observado{}}
}

// livro:inicio membro

// Run bate e observa a cada intervalo, até ctx terminar. Um erro de
// banco não para o laço: o nó que não consegue bater vai ser suspeito
// para os outros, e é isso que ele deve ser.
func (m *Membro) Run(ctx context.Context) error {
	if _, err := m.cfg.DB.Exec(ctx, `INSERT INTO cluster_member
		(node, addr) VALUES ($1, $2) ON CONFLICT (node) DO UPDATE
		SET addr = $2, started_at = now(), beat_at = now()`,
		m.cfg.No, m.cfg.Endereco); err != nil {
		return err
	}
	t := time.NewTicker(m.cfg.Intervalo)
	defer t.Stop()
	for {
		if err := m.ciclo(ctx); err != nil && ctx.Err() == nil {
			m.cfg.Log.WarnContext(ctx, "membership: ciclo falhou",
				slog.Any("erro", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// ciclo bate e lê a tabela inteira, com o now() do banco. Cada beat_at
// novo é uma batida para o detector daquele nó; o now() é o instante
// em que todos os detectores são consultados. Nenhum relógio de nó
// entra na conta.
func (m *Membro) ciclo(ctx context.Context) error {
	if _, err := m.cfg.DB.Exec(ctx, `UPDATE cluster_member
		SET beat_at = now() WHERE node = $1`, m.cfg.No); err != nil {
		return err
	}
	rows, err := m.cfg.DB.Query(ctx, `SELECT node, addr, started_at,
		beat_at, now() FROM cluster_member WHERE node <> $1`, m.cfg.No)
	if err != nil {
		return err
	}
	defer rows.Close()
	m.mu.Lock()
	defer m.mu.Unlock()
	for rows.Next() {
		var (
			no                    coordinator.NodeID
			end                   string
			inicio, batida, agora time.Time
		)
		if err := rows.Scan(&no, &end, &inicio, &batida,
			&agora); err != nil {
			return err
		}
		o := m.outros[no]
		if o == nil || !o.inicio.Equal(inicio) {
			o = &observado{detector: m.cfg.Detector(), inicio: inicio}
			m.outros[no] = o
		}
		if !o.ultima.Equal(batida) {
			o.detector.Batida(batida)
			o.ultima = batida
		}
		o.endereco, m.agora = end, agora
	}
	if err := rows.Err(); err != nil {
		return err
	}
	m.viu = true
	return nil
}

// livro:fim membro

// No é o nome deste nó.
func (m *Membro) No() coordinator.NodeID { return m.cfg.No }

// Members devolve este nó e os que ele não dá por mortos, em ordem.
// Antes da primeira leitura da tabela, não sabe — e diz que não sabe:
// um nó que acabou de subir e respondesse "só eu" levaria um líder a
// dar todas as partições a um nó só (Cap. 32).
func (m *Membro) Members(
	context.Context,
) ([]coordinator.NodeID, error) {
	m.mu.Lock()
	viu := m.viu
	m.mu.Unlock()
	if !viu {
		return nil, coordinator.ErrSemVisao
	}
	ids := []coordinator.NodeID{m.cfg.No}
	for _, s := range m.Visao() {
		if s.Estado != Morto {
			ids = append(ids, s.No)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

// Visao devolve o que este nó acha de cada um dos outros.
func (m *Membro) Visao() []Situacao {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s []Situacao
	for no, o := range m.outros {
		sit := Situacao{No: no, Endereco: o.endereco,
			Estado:   o.detector.Estado(m.agora),
			Silencio: m.agora.Sub(o.ultima)}
		if p, ok := o.detector.(*Phi); ok {
			sit.Phi = p.Valor(m.agora)
		}
		s = append(s, sit)
	}
	slices.SortFunc(s, func(a, b Situacao) int {
		return cmp.Compare(a.No, b.No)
	})
	return s
}
