//go:build !langforge_generated

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "vehicle-report-demo requires generated LangForge output; run `make -C examples/go/vehicle-report run`")
	os.Exit(2)
}
