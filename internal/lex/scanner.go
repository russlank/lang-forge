package lex

import (
	"fmt"
	"unicode/utf8"

	"github.com/russlank/lang-forge/internal/spec"
)

// ScannerConfig is the lexer-runtime contract used by in-process and generated
// scanners. The domain is sparse so UTF-8 support does not require dense
// Unicode-sized transition tables.
type ScannerConfig struct {
	Encoding string   `json:"encoding"`
	Invalid  string   `json:"invalid"`
	Newline  string   `json:"newline,omitempty"`
	Domain   RangeSet `json:"domain"`
}

// DefaultScannerConfig returns LangForge's UTF-8 scanner contract.
func DefaultScannerConfig() ScannerConfig {
	return ScannerConfig{
		Encoding: string(spec.ScannerEncodingUTF8),
		Invalid:  string(spec.ScannerInvalidError),
		Domain:   UnicodeScalarDomain(),
	}
}

func scannerConfigFromSpec(scanner spec.ScannerSpec) (ScannerConfig, error) {
	scanner = scanner.WithDefaults()
	if scanner.Encoding != spec.ScannerEncodingUTF8 {
		return ScannerConfig{}, fmt.Errorf("unsupported scanner encoding %q", scanner.Encoding)
	}
	if scanner.Invalid != spec.ScannerInvalidError {
		return ScannerConfig{}, fmt.Errorf("unsupported scanner invalid-input policy %q", scanner.Invalid)
	}
	if scanner.Newline != "" && scanner.Newline != "lf" {
		return ScannerConfig{}, fmt.Errorf("unsupported scanner newline policy %q", scanner.Newline)
	}
	cfg := DefaultScannerConfig()
	cfg.Newline = scanner.Newline
	return cfg, nil
}

func decodeUTF8ScannerRune(input string, pos int) (rune, int, error) {
	if pos >= len(input) {
		return 0, 0, fmt.Errorf("invalid scanner offset %d", pos)
	}
	r, size := utf8.DecodeRuneInString(input[pos:])
	if r == utf8.RuneError && size == 1 && input[pos] >= utf8.RuneSelf {
		return 0, 0, fmt.Errorf("invalid UTF-8 at byte %d", pos)
	}
	if !IsUnicodeScalar(r) {
		return 0, 0, fmt.Errorf("invalid Unicode scalar at byte %d", pos)
	}
	return r, size, nil
}

func advanceScalarPosition(input string, start, end, line, column int) (int, int) {
	for pos := start; pos < end; {
		r, size, err := decodeUTF8ScannerRune(input, pos)
		if err != nil {
			return line, column
		}
		pos += size
		if r == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}
