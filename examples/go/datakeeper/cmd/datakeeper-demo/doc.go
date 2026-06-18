//go:build !langforge_generated

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "datakeeper-demo requires generated LangForge output; run `make -C examples/go/datakeeper run`")
	os.Exit(2)
}
