//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	calc "github.com/russlank/lang-forge/examples/go/calc/generated"
	calcsem "github.com/russlank/lang-forge/examples/go/calc/semantics"
)

func main() {
	inputPath := flag.String("input", "examples/go/calc/input.calc", "calculator expression file")
	logPath := flag.String("log", "", "optional report file")
	flag.Parse()

	source, err := os.ReadFile(*inputPath)
	if err != nil {
		exitf("read input: %v", err)
	}
	report, err := runCalcDemo(*inputPath, string(source))
	if err != nil {
		exitf("%v", err)
	}
	fmt.Print(report)
	if *logPath != "" {
		if err := os.WriteFile(*logPath, []byte(report), 0o644); err != nil {
			exitf("write log: %v", err)
		}
		fmt.Fprintf(os.Stderr, "wrote report to %s\n", *logPath)
	}
}

func runCalcDemo(name, source string) (string, error) {
	value, err := calc.ParseWithReducerFromSource(calc.NewScanner(source), calc.ReducerFunc(calcsem.Reduce))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", name, err)
	}
	result, ok := value.(float64)
	if !ok {
		return "", fmt.Errorf("parse %s: semantic result has type %T", name, value)
	}
	tokens, err := calc.Tokenize(source)
	if err != nil {
		return "", fmt.Errorf("tokenize %s for report: %w", name, err)
	}

	var report bytes.Buffer
	fmt.Fprintf(&report, "Calc LangForge demo: %s\n", name)
	fmt.Fprintln(&report, strings.Repeat("=", 72))
	fmt.Fprintln(&report)
	fmt.Fprintln(&report, "Input")
	fmt.Fprintf(&report, "  %s\n", strings.TrimSpace(source))
	fmt.Fprintln(&report)
	fmt.Fprintln(&report, "Token stream")
	for i, token := range tokens {
		fmt.Fprintf(&report, "  %03d  %-8s  %q  [%d:%d]\n", i, token.Token, token.Text, token.Start, token.End)
	}
	fmt.Fprintln(&report)
	fmt.Fprintln(&report, "Parse result: ok")
	fmt.Fprintf(&report, "Semantic result: %g\n", result)
	return report.String(), nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
