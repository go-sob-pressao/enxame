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
	// emSistema, se não for nil, limita os itens em qualquer lugar da
	// fila: no canal, em processamento ou esperando para retentar.
	emSistema chan struct{}
}

// Nova cria a fila com vagas no canal.
func Nova(vagas int) *Fila {
	return &Fila{canal: make(chan Item, vagas)}
}

// Oferecer põe o item no canal, se houver vaga.
func (f *Fila) Oferecer(it Item) bool {
	if f.emSistema != nil {
		select {
		case f.emSistema <- struct{}{}:
		default:
			return false
		}
	}
	select {
	case f.canal <- it:
		return true
	default:
		if f.emSistema != nil {
			<-f.emSistema
		}
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
			return
		}
		if f.emSistema != nil {
			<-f.emSistema
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

// Limitada é a correção: o limite vale para o sistema inteiro, e um
// item só devolve a vaga quando termina.
func Limitada(vagas, total int) *Fila {
	f := Nova(vagas)
	f.emSistema = make(chan struct{}, total)
	return f
}

// Tamanho devolve os itens no canal e os que esperam para retentar.
func (f *Fila) Tamanho() (canal, retentar int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.canal), len(f.retentar)
}
