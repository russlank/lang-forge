//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/russlank/lang-forge/examples/go/draw"
)

func main() {
	inputPath := flag.String("input", "examples/go/draw/sample.draw", "DRAW source file")
	outputPath := flag.String("output", "examples/go/draw/dist/sample.png", "PNG output file")
	logPath := flag.String("log", "", "optional report file")
	flag.Parse()

	source, err := os.ReadFile(*inputPath)
	if err != nil {
		exitf("read input: %v", err)
	}
	program, err := draw.Compile(string(source))
	if err != nil {
		exitf("compile: %v", err)
	}
	if err := os.MkdirAll(parentDir(*outputPath), 0o755); err != nil {
		exitf("create output directory: %v", err)
	}
	result, err := draw.RenderPNG(program, *outputPath)
	if err != nil {
		exitf("render: %v", err)
	}
	var report bytes.Buffer
	draw.WriteReport(&report, *inputPath, *outputPath, result)
	fmt.Print(report.String())
	if *logPath != "" {
		if err := os.WriteFile(*logPath, report.Bytes(), 0o644); err != nil {
			exitf("write log: %v", err)
		}
		fmt.Fprintf(os.Stderr, "wrote report to %s\n", *logPath)
	}
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
