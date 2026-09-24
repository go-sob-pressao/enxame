// Command deferlaco: o defer que só executa quando a função termina.
package main

import "fmt"

func main() {
	p := &Pool{}
	processarTodos(p, 1000)
	fmt.Println(
		"pico de recursos abertos ao mesmo tempo:",
		p.pico.Load(),
	)
}
