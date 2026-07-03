//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/russlank/lang-forge/examples/templates/go/library-dsl/model"
	"github.com/russlank/lang-forge/examples/templates/go/library-dsl/parser"
)

func main() {
	inputPath := flag.String("input", "input.dsl", "DSL source file")
	logPath := flag.String("log", "", "optional report file")
	flag.Parse()

	source, err := os.ReadFile(*inputPath)
	if err != nil {
		exitf("read input: %v", err)
	}
	document, err := parser.New().Parse(string(source))
	if err != nil {
		exitf("parse: %s", parser.FormatError(err))
	}
	report := buildReport(*inputPath, document)
	fmt.Print(report)
	if *logPath != "" {
		if err := os.WriteFile(*logPath, []byte(report), 0o644); err != nil {
			exitf("write log: %v", err)
		}
	}
}

func buildReport(inputPath string, document model.Document) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Library DSL Go template: %s\n", inputPath)
	for _, entry := range document.Entries {
		fmt.Fprintf(&b, "  %s %s = %s\n", entry.Kind, entry.Name, entry.Value.String())
	}
	return b.String()
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
