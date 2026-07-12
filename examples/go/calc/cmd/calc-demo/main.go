//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	calc "github.com/russlank/lang-forge/examples/go/calc/generated"
	calcsem "github.com/russlank/lang-forge/examples/go/calc/semantics"
)

func main() {
	inputPath := flag.String("input", "examples/go/calc/input.calc", "calculator expression file")
	logPath := flag.String("log", "", "optional report file")
	flag.Parse()

	report, err := runCalcDemoFile(*inputPath)
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
	return runCalcDemoFromReaders(name, strings.NewReader(source), strings.NewReader(source), source)
}

func runCalcDemoFile(path string) (string, error) {
	parseInput, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open input for parse: %w", err)
	}
	defer parseInput.Close()

	source, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read input for report: %w", err)
	}
	return runCalcDemoFromReaders(path, parseInput, bytes.NewReader(source), string(source))
}

func runCalcDemoFromReaders(name string, parseInput io.Reader, tokenInput io.Reader, source string, options ...calc.ReaderScannerOption) (string, error) {
	// Production-style parsing uses the reader-backed scanner. The parser pulls
	// lexemes lazily, so file, stdin, pipe, and virtual inputs do not need to be
	// materialized just to drive parsing. The second reader is used only for the
	// teaching report that prints the full token stream.
	value, err := calc.ParseWithReducerFromLexemeSource(calc.NewReaderScanner(parseInput, options...), calc.ReducerFunc(calcsem.Reduce))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", name, err)
	}
	result, ok := value.(float64)
	if !ok {
		return "", fmt.Errorf("parse %s: semantic result has type %T", name, value)
	}
	tokens, err := calc.TokenizeFromReader(tokenInput, options...)
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
