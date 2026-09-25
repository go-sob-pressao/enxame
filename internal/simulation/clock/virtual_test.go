package clock_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/simulation/clock"
)

var t0 = time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)

func TestVirtualDisparaEmOrdemDePrazo(t *testing.T) {
	v := clock.NewVirtual(t0)
	tres := v.NewTimer(3 * time.Second)
	um := v.NewTimer(time.Second)
	cancelado := v.NewTimer(2 * time.Second)
	if !cancelado.Stop() {
		t.Fatal("Stop antes do prazo deveria devolver true")
	}
	v.Advance(5 * time.Second)
	if got := <-um.C(); !got.Equal(t0.Add(time.Second)) {
		t.Errorf("um disparou em %v", got)
	}
	if got := <-tres.C(); !got.Equal(t0.Add(3 * time.Second)) {
		t.Errorf("tres disparou em %v", got)
	}
	select {
	case <-cancelado.C():
		t.Error("timer parado disparou")
	default:
	}
	if !v.Now().Equal(t0.Add(5 * time.Second)) {
		t.Errorf("relógio em %v", v.Now())
	}
}

func TestVirtualNaoAndaSozinho(t *testing.T) {
	v := clock.NewVirtual(t0)
	tm := v.NewTimer(time.Nanosecond)
	select {
	case <-tm.C():
		t.Fatal("disparou sem Advance")
	default:
	}
	v.Advance(time.Nanosecond)
	<-tm.C()
}

func TestVirtualNaoVoltaNoTempo(t *testing.T) {
	v := clock.NewVirtual(t0)
	v.Advance(time.Minute)
	atrasado := v.NewTimer(-time.Second) // prazo já vencido
	v.Advance(time.Second)
	if got := <-atrasado.C(); !got.Equal(t0.Add(time.Minute)) {
		t.Fatalf("disparou em %v; o relógio voltou", got)
	}
	if !v.Now().Equal(t0.Add(time.Minute + time.Second)) {
		t.Fatalf("relógio em %v", v.Now())
	}
}
