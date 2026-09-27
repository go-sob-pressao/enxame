//go:build chaos

// Package chaos — experimentos de caos contra o cluster real: três
// enxamed de verdade, o PostgreSQL de verdade, e falhas provocadas no
// caminho entre eles (Capítulo 28).
package chaos

import (
	"context"
	"net"
	"sync"
	"time"
)

// livro:inicio proxy

// Proxy fica entre um enxamed e o PostgreSQL e decide o que passa. Cada
// nó conecta no banco pela porta do seu proxy; cortar o proxy é isolar
// o nó do banco, e só ele. Atrasar é pôr latência em cada pedaço que
// atravessa, nos dois sentidos.
type Proxy struct {
	Destino string // o PostgreSQL

	lis     net.Listener
	mu      sync.Mutex
	cortado bool
	atraso  time.Duration
	conns   map[net.Conn]struct{}
}

// Cortar derruba as conexões abertas e recusa as novas: para o nó, o
// banco sumiu. Religar desfaz.
func (p *Proxy) Cortar() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cortado = true
	for c := range p.conns {
		_ = c.Close()
	}
}

// Religar volta a deixar tudo passar.
func (p *Proxy) Religar() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cortado = false
}

// Atrasar põe d de latência em cada sentido; zero tira.
func (p *Proxy) Atrasar(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.atraso = d
}

// livro:fim proxy

// Abrir começa a escutar numa porta livre e devolve o endereço.
func (p *Proxy) Abrir() (string, error) {
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	p.lis, p.conns = lis, map[net.Conn]struct{}{}
	go p.aceitar()
	return lis.Addr().String(), nil
}

// Fechar para de escutar e derruba tudo.
func (p *Proxy) Fechar() {
	_ = p.lis.Close()
	p.Cortar()
}

func (p *Proxy) aceitar() {
	for {
		c, err := p.lis.Accept()
		if err != nil {
			return
		}
		p.mu.Lock()
		cortado := p.cortado
		p.mu.Unlock()
		if cortado {
			_ = c.Close()
			continue
		}
		var d net.Dialer
		b, err := d.DialContext(context.Background(), "tcp", p.Destino)
		if err != nil {
			_ = c.Close()
			continue
		}
		p.mu.Lock()
		p.conns[c], p.conns[b] = struct{}{}, struct{}{}
		p.mu.Unlock()
		go p.copiar(b, c)
		go p.copiar(c, b)
	}
}

// copiar leva os bytes de um lado ao outro, com o atraso da vez.
func (p *Proxy) copiar(para, de net.Conn) {
	defer func() {
		_ = para.Close()
		_ = de.Close()
		p.mu.Lock()
		delete(p.conns, para)
		delete(p.conns, de)
		p.mu.Unlock()
	}()
	buf := make([]byte, 32*1024)
	for {
		n, err := de.Read(buf)
		if n > 0 {
			p.mu.Lock()
			d := p.atraso
			p.mu.Unlock()
			if d > 0 {
				time.Sleep(d)
			}
			if _, werr := para.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil { // io.EOF ou conexão derrubada: acabou
			return
		}
	}
}
