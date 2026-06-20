//go:build langforge_generated

package main

import (
	"strings"
	"testing"
)

func TestRunCalcDemoAcceptsSample(t *testing.T) {
	report, err := runCalcDemo("sample", "1 + 2 * (3 - 4.5)")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "Parse result: ok") ||
		!strings.Contains(report, "Semantic result: -2") ||
		!strings.Contains(report, "Number") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestRunCalcDemoRejectsMalformedExpression(t *testing.T) {
	_, err := runCalcDemo("bad", "1+")
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestRunCalcDemoRejectsDivisionByZero(t *testing.T) {
	_, err := runCalcDemo("bad", "1/0")
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("error = %v, want division-by-zero error", err)
	}
}

func TestRunCalcDemoRejectsUnmatchedInput(t *testing.T) {
	_, err := runCalcDemo("bad", "1@2")
	if err == nil || !strings.Contains(err.Error(), "tokenize") {
		t.Fatalf("error = %v, want tokenize error", err)
	}
}
