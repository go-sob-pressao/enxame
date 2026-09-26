package main

import (
	"encoding/json/v2"
	"fmt"
	"text/tabwriter"
	"time"
)

// livro:inicio ctl-cluster

// clusterMembers mostra quem está no cluster na visão do nó que
// respondeu. Perguntar a outro nó pode dar outra resposta.
func clusterMembers(c *cliente, args []string) error {
	fs := novoFlagSet("cluster members")
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, err := c.fazer("GET", "/v1/cluster/members", nil)
	if err != nil {
		return err
	}
	var lista struct {
		Items []struct {
			No       string  `json:"node"`
			Estado   string  `json:"state"`
			Phi      float64 `json:"phi"`
			Silencio int64   `json:"silence_ms"`
			Este     bool    `json:"self"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &lista); err != nil {
		return err
	}
	w := tabwriter.NewWriter(c.saida, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NÓ\tESTADO\tPHI\tSILÊNCIO")
	for _, m := range lista.Items {
		if m.Este {
			fmt.Fprintf(w, "%s\t%s\t—\t— (este nó)\n", m.No, m.Estado)
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%.1f\t%v\n", m.No, m.Estado, m.Phi,
			time.Duration(m.Silencio)*time.Millisecond)
	}
	return w.Flush()
}

// livro:fim ctl-cluster
