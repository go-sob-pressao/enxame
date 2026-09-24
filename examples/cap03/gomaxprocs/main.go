// Command gomaxprocs: o que o runtime enxerga dentro de um contêiner.
//
// A partir do Go 1.25, em Linux, GOMAXPROCS respeita o limite de CPU do
// cgroup. Compare:
//
//	GOOS=linux go build -o /tmp/gmp ./examples/cap03/gomaxprocs
//	docker run --rm --cpus=2 -v /tmp/gmp:/gmp busybox /gmp
//	    NumCPU=8 GOMAXPROCS=2
//	docker run --rm --cpus=2 -e GODEBUG=containermaxprocs=0 \
//	    -v /tmp/gmp:/gmp busybox /gmp
//	    NumCPU=8 GOMAXPROCS=8   (como antes do Go 1.25)
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf(
		"NumCPU=%d GOMAXPROCS=%d\n",
		runtime.NumCPU(),
		runtime.GOMAXPROCS(0),
	)
}
