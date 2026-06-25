//go:build !langforge_generated

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "parser-recovery-demo requires generated output; run `make -C examples/go/parser-recovery run`")
	os.Exit(2)
}
