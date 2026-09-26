package http

import (
	"net/http"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
)

// Cluster é o que a API sabe do cluster: a visão deste nó.
type Cluster interface {
	No() coordinator.NodeID
	Visao() []membership.Situacao
}

// MembroCluster é um nó como este nó o vê.
type MembroCluster struct {
	No         string  `json:"node"`
	Endereco   string  `json:"address,omitzero"`
	Estado     string  `json:"state"`
	Phi        float64 `json:"phi"`
	SilencioMs int64   `json:"silence_ms"`
	Este       bool    `json:"self,omitzero"`
}

// membrosCluster responde a visão deste nó: ele mesmo e os outros, com
// o estado, o phi e o silêncio de cada um. Outro nó pode responder
// diferente — é uma visão, não um fato.
func (a *API) membrosCluster(w http.ResponseWriter, _ *http.Request) {
	lista := Lista[MembroCluster]{Items: []MembroCluster{}}
	if a.Cluster != nil {
		lista.Items = append(lista.Items, MembroCluster{
			No: string(a.Cluster.No()), Estado: "vivo", Este: true})
		for _, s := range a.Cluster.Visao() {
			lista.Items = append(lista.Items, MembroCluster{
				No: string(s.No), Endereco: s.Endereco,
				Estado: s.Estado.String(), Phi: min(s.Phi, 300),
				SilencioMs: s.Silencio.Milliseconds()})
		}
	}
	escrever(w, http.StatusOK, lista)
}
