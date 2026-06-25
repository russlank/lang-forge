//go:build langforge_generated

package main

import (
	"flag"
	"fmt"
	"os"

	recovery "github.com/russlank/lang-forge/examples/go/parser-recovery/generated"
)

func main() {
	inputPath := flag.String("input", "input.recovery", "source file containing recoverable statement errors")
	assert := flag.Bool("assert", false, "verify the teaching fixture's expected recovery result")
	flag.Parse()

	source, err := os.ReadFile(*inputPath)
	if err != nil {
		exitf("read input: %v", err)
	}
	result, err := parseSource(string(source))
	if err != nil {
		exitf("parse input: %v", err)
	}

	fmt.Printf("accepted: %t\n", result.Accepted)
	for index, diagnostic := range result.Diagnostics {
		fmt.Printf(
			"%d. %d:%d unexpected %s; expected %s; recovery=%s discarded=%d\n",
			index+1,
			diagnostic.StartLine,
			diagnostic.StartColumn,
			diagnostic.UnexpectedDisplay,
			expectedDisplay(diagnostic.Expected),
			diagnostic.Recovery.Kind,
			diagnostic.Recovery.Discarded,
		)
	}
	if *assert {
		if !result.Accepted || len(result.Diagnostics) != 2 {
			exitf("fixture result = accepted:%t diagnostics:%d, want accepted:true diagnostics:2", result.Accepted, len(result.Diagnostics))
		}
	}
}

func parseSource(source string) (recovery.ParseResult, error) {
	tokens, err := recovery.Tokenize(source)
	if err != nil {
		return recovery.ParseResult{}, err
	}
	return recovery.ParseRecovering(tokens)
}

func expectedDisplay(expected []recovery.ExpectedToken) string {
	if len(expected) == 0 {
		return "<none>"
	}
	out := expected[0].Display
	for _, token := range expected[1:] {
		out += ", " + token.Display
	}
	return out
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
