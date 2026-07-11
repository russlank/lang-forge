//go:build langforge_generated

package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	calc "github.com/russlank/lang-forge/examples/go/calc/generated"
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

func TestRunCalcDemoAcceptsChunkedReaders(t *testing.T) {
	source := "1 + 2 * (3 - 4.5)"
	report, err := runCalcDemoFromReaders(
		"chunked",
		strings.NewReader(source),
		strings.NewReader(source),
		source,
		calc.WithReaderScannerBufferSize(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "Semantic result: -2") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestRunCalcDemoRejectsMalformedExpression(t *testing.T) {
	_, err := runCalcDemo("bad", "1+")
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestRunCalcDemoRejectsReaderFailure(t *testing.T) {
	readErr := errors.New("reader failed")
	_, err := runCalcDemoFromReaders(
		"bad-reader",
		&failingReader{chunks: []string{"1", " + "}, err: readErr},
		strings.NewReader("1 + 2"),
		"1 + 2",
		calc.WithReaderScannerBufferSize(1),
	)
	if !errors.Is(err, readErr) {
		t.Fatalf("error = %v, want reader failure", err)
	}
}

func TestRunCalcDemoRejectsBufferedTokenLimit(t *testing.T) {
	_, err := runCalcDemoFromReaders(
		"too-long",
		strings.NewReader("123"),
		strings.NewReader("123"),
		"123",
		calc.WithReaderScannerBufferSize(1),
		calc.WithMaxBufferedLexemeBytes(2),
	)
	if err == nil || !strings.Contains(err.Error(), "buffered lexeme exceeds") {
		t.Fatalf("error = %v, want buffered-lexeme limit error", err)
	}
}

type failingReader struct {
	chunks []string
	err    error
}

func (r *failingReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, r.err
	}
	n := copy(p, r.chunks[0])
	r.chunks[0] = r.chunks[0][n:]
	if r.chunks[0] == "" {
		r.chunks = r.chunks[1:]
	}
	if n == 0 {
		return 0, io.ErrNoProgress
	}
	return n, nil
}

func TestRunCalcDemoRejectsDivisionByZero(t *testing.T) {
	_, err := runCalcDemo("bad", "1/0")
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("error = %v, want division-by-zero error", err)
	}
}

func TestRunCalcDemoRejectsUnmatchedInput(t *testing.T) {
	_, err := runCalcDemo("bad", "1@2")
	if err == nil || !strings.Contains(err.Error(), "parse") || !strings.Contains(err.Error(), "lexical") && !strings.Contains(err.Error(), "no lexical rule") {
		t.Fatalf("error = %v, want streaming scanner error", err)
	}
}
