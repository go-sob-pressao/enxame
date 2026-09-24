// Command appendaliasing: dois slices, um array.
package main

import "fmt"

func main() {
	a, b := derivar([]int{1, 2, 3})
	fmt.Println("a:", a, "b:", b)
}
