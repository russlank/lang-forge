//go:build langforge_generated

package main

import (
	"os"
	"testing"
)

func TestTeachingFixtureRecoversTwice(t *testing.T) {
	source, err := os.ReadFile("../../input.recovery")
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseSource(string(source))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted || len(result.Diagnostics) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if got := result.Diagnostics[0].Expected[0].Display; got != "number literal" {
		t.Fatalf("expected display = %q", got)
	}
	if got := result.Diagnostics[1].StartLine; got != 3 {
		t.Fatalf("second diagnostic line = %d", got)
	}
}
