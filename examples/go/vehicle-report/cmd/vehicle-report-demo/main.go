//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	vehiclereport "github.com/russlank/lang-forge/examples/go/vehicle-report"
)

func main() {
	inputPath := flag.String("input", "sample.vehicle", "vehicle source file")
	logPath := flag.String("log", "", "optional report file")
	flag.Parse()

	source, err := os.ReadFile(*inputPath)
	if err != nil {
		exitf("read input: %v", err)
	}
	vehicle, err := vehiclereport.Parse(string(source))
	if err != nil {
		exitf("parse input: %v", err)
	}
	var report bytes.Buffer
	vehiclereport.WriteReport(&report, *inputPath, vehicle)
	fmt.Print(report.String())
	if *logPath != "" {
		if err := os.WriteFile(*logPath, report.Bytes(), 0o644); err != nil {
			exitf("write log: %v", err)
		}
		fmt.Fprintf(os.Stderr, "wrote report to %s\n", *logPath)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
