package main

// As regras ficam em slices, não em maps: a ordem de avaliação e a
// ordem da saída são sempre as mesmas. Num livro sobre determinismo, a
// ferramenta que guarda a arquitetura não pode imprimir em ordem
// aleatória.

// regraDeCamada proíbe que os pacotes sob Camada importem pacotes sob
// qualquer um dos Proibidos. Uma camada sempre pode importar a si
// mesma.
type regraDeCamada struct {
	Camada    string
	Proibidos []string
}

var regrasDeCamada = []regraDeCamada{
	// O domínio é puro: não conhece nenhuma outra parte do sistema.
	{"internal/core", []string{"internal"}},
	{
		"internal/engine",
		[]string{
			"internal/transport",
			"internal/worker",
			"internal/cluster/raft",
			"internal/delivery",
		},
	},
	{
		"internal/queue",
		[]string{
			"internal/transport",
			"internal/store/postgres",
			"internal/store/sqlite",
			"internal/cluster",
		},
	},
	{
		"internal/delivery",
		[]string{
			"internal/store/postgres",
			"internal/store/sqlite",
			"internal/cluster",
			"internal/transport/grpc",
		},
	},
	{
		"internal/store",
		[]string{
			"internal/engine",
			"internal/queue",
			"internal/delivery",
			"internal/transport",
			"internal/worker",
			"internal/cluster",
		},
	},
	// O contrato não conhece as implementações: é o consumidor que o
	// declara.
	{
		"internal/cluster/coordinator",
		[]string{
			"internal/cluster/pgcoord",
			"internal/cluster/raftcoord",
		},
	},
	// Raft é um algoritmo; não sabe que existe banco nem motor.
	{
		"internal/cluster/raft",
		[]string{
			"internal/store",
			"internal/engine",
			"internal/queue",
			"internal/cluster/pgcoord",
		},
	},
	{
		"internal/simulation",
		[]string{
			"internal/transport/grpc",
			"internal/store/postgres",
			"internal/store/sqlite",
		},
	},
	{
		"pkg",
		[]string{
			"internal/engine",
			"internal/cluster",
			"internal/simulation",
			"internal/transport",
		},
	},
}

// livro:inicio pureza

// camadaPura é a camada que, além das regras acima, não pode tocar o
// mundo: nem biblioteca de terceiros, nem I/O da stdlib, nem relógio,
// nem acaso.
const camadaPura = "internal/core"

// stdlibProibidaNoCore lista prefixos da stdlib que fazem I/O ou
// introduzem não determinismo. time continua permitido (time.Time e
// time.Duration são valores); o que é proibido são as funções que leem
// o relógio.
var stdlibProibidaNoCore = []string{
	"crypto/rand",
	"database",
	"embed",
	"io/fs",
	"log",
	"math/rand",
	"net",
	"os",
	"plugin",
	"runtime",
	"syscall",
	"unsafe",
}

// funcaoProibidaNoCore identifica uma função da stdlib que o domínio
// não pode chamar, e o que usar no lugar.
type funcaoProibidaNoCore struct {
	Pacote     string
	Nomes      []string
	Substituto string
}

var funcoesProibidasNoCore = []funcaoProibidaNoCore{
	{
		"time",
		[]string{
			"After",
			"AfterFunc",
			"NewTicker",
			"NewTimer",
			"Now",
			"Since",
			"Sleep",
			"Tick",
			"Until",
		},
		"receba o instante ou um Clock como parâmetro",
	},
	{
		"uuid",
		[]string{"New", "NewV4", "NewV7"},
		"receba o identificador pronto da borda",
	},
	// Os construtores de prazo do context leem o relógio por dentro:
	// context.WithTimeout(ctx, d) é time.Now().Add(d) com outro nome.
	{
		"context",
		[]string{
			"WithDeadline",
			"WithDeadlineCause",
			"WithTimeout",
			"WithTimeoutCause",
		},
		"o prazo é assunto da borda; receba o instante",
	},
}

// livro:fim pureza
