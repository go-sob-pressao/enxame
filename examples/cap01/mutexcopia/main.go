// Command mutexcopia: o Mutex copiado por valor não protege nada.
package main

import "fmt"

func main() {
	fmt.Println("total:", somar(1000), "(esperado 1000)")
}
