// Package retentar é o enigma do Capítulo 21: a fila limitada a 10 mil
// que nunca passava de 10 mil — e o processo que morria sem memória.
package retentar

import "sync"

// Item é o trabalho, com o corpo que ele carrega.
type Item struct {
	ID    int
	Corpo []byte
}

// livro:inicio enigma

// Fila tem um canal de vagas limitadas: quem oferece com ele cheio
// ouve "não", e a pressão volta para quem produz. O que falha no
// processamento vai para retentar e volta ao canal mais tarde, quando
// Reenviar roda e há vaga.
type Fila struct {
	canal    chan Item
	mu       sync.Mutex
	retentar []Item
}

// Oferecer põe o item no canal, se houver vaga.
func (f *Fila) Oferecer(it Item) bool {
	select {
	case f.canal <- it:
		return true
	default:
		return false
	}
}

// Processar tira um item do canal e o entrega a h. Se h falha, o item
// vai para retentar.
func (f *Fila) Processar(h func(Item) error) {
	select {
	case it := <-f.canal:
		if err := h(it); err != nil {
			f.mu.Lock()
			f.retentar = append(f.retentar, it)
			f.mu.Unlock()
		}
	default:
	}
}

// Reenviar devolve ao canal os itens que esperam, enquanto houver vaga.
func (f *Fila) Reenviar() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for len(f.retentar) > 0 {
		select {
		case f.canal <- f.retentar[0]:
			f.retentar = f.retentar[1:]
		default:
			return
		}
	}
}

// livro:fim enigma

// Nova cria a fila com vagas no canal.
func Nova(vagas int) *Fila {
	return &Fila{canal: make(chan Item, vagas)}
}

// Tamanho devolve os itens no canal e os que esperam para retentar.
func (f *Fila) Tamanho() (canal, retentar int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.canal), len(f.retentar)
}

// livro:inicio correcao

// Limitada põe o limite no sistema inteiro: a vaga é tomada na
// admissão e só volta quando o item termina com sucesso — esperar para
// retentar não devolve a vaga.
type Limitada struct {
	*Fila
	emSistema chan struct{}
}

// NovaLimitada cria a fila com vagas no canal e total no sistema.
func NovaLimitada(vagas, total int) *Limitada {
	return &Limitada{Nova(vagas), make(chan struct{}, total)}
}

// Oferecer admite o item só se houver vaga no sistema e no canal.
func (l *Limitada) Oferecer(it Item) bool {
	select {
	case l.emSistema <- struct{}{}:
	default:
		return false
	}
	if !l.Fila.Oferecer(it) {
		<-l.emSistema
		return false
	}
	return true
}

// Processar devolve a vaga do sistema quando h termina sem erro.
func (l *Limitada) Processar(h func(Item) error) {
	l.Fila.Processar(func(it Item) error {
		err := h(it)
		if err == nil {
			<-l.emSistema
		}
		return err
	})
}

// livro:fim correcao
