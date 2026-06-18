// Package parseralgo centralizes supported parser table construction modes.
package parseralgo

import (
	"strings"
)

const (
	// SLR selects an SLR(1) table, mostly useful for compatibility checks.
	SLR = "slr"
	// LALR selects the default LALR(1) table used by classic Yacc-like tools.
	LALR = "lalr"
	// IELR selects a deterministic LR(1) table that preserves canonical LR
	// language recognition while merging states where doing so is safe.
	IELR = "ielr"
	// Canonical selects the full canonical LR(1) table.
	Canonical = "canonical"
	// Default is the parser algorithm used when a specification omits %type.
	Default = LALR
)

// Allowed returns a human-readable list of accepted algorithm names.
func Allowed() string {
	return SLR + ", " + LALR + ", " + IELR + ", or " + Canonical
}

// Parse validates a non-empty parser algorithm directive value.
func Parse(value string) (string, bool) {
	algorithm := strings.ToLower(strings.TrimSpace(value))
	switch algorithm {
	case SLR, LALR, IELR, Canonical:
		return algorithm, true
	default:
		return "", false
	}
}

// Normalize resolves an optional algorithm value to a concrete supported mode.
func Normalize(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return Default, true
	}
	return Parse(value)
}
