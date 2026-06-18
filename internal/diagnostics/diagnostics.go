package diagnostics

import (
	"fmt"
	"strings"
)

// Severity describes how a diagnostic should affect command execution.
type Severity string

const (
	// Error blocks validation, table construction, or generation.
	Error Severity = "error"
	// Warning reports a recoverable issue.
	Warning Severity = "warning"
	// Info reports contextual information.
	Info Severity = "info"
)

// Position is a 1-based source location plus its original byte offset.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Offset int `json:"offset"`
}

// Span identifies a half-open source range.
type Span struct {
	File  string   `json:"file,omitempty"`
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// String formats the start of the span for command-line diagnostics.
func (s Span) String() string {
	if s.File == "" {
		return fmt.Sprintf("%d:%d", s.Start.Line, s.Start.Column)
	}
	return fmt.Sprintf("%s:%d:%d", s.File, s.Start.Line, s.Start.Column)
}

// Diagnostic is one validation or build message.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code,omitempty"`
	Message  string   `json:"message"`
	Span     Span     `json:"span"`
}

// Error formats the diagnostic as a human-readable message.
func (d Diagnostic) Error() string {
	if d.Code == "" {
		return fmt.Sprintf("%s: %s: %s", d.Span, d.Severity, d.Message)
	}
	return fmt.Sprintf("%s: %s %s: %s", d.Span, d.Severity, d.Code, d.Message)
}

// List is an ordered collection of diagnostics.
type List []Diagnostic

// Error formats all diagnostics in the list, one per line.
func (l List) Error() string {
	if len(l) == 0 {
		return ""
	}
	var b strings.Builder
	for i, d := range l {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(d.Error())
	}
	return b.String()
}

// HasErrors reports whether the list contains a blocking error.
func (l List) HasErrors() bool {
	for _, d := range l {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// Add appends a diagnostic with the supplied severity.
func (l *List) Add(sev Severity, code, msg string, span Span) {
	*l = append(*l, Diagnostic{Severity: sev, Code: code, Message: msg, Span: span})
}

// AddError appends a blocking error diagnostic.
func (l *List) AddError(code, msg string, span Span) {
	l.Add(Error, code, msg, span)
}

// Source maps byte offsets back to line and column positions.
type Source struct {
	File  string
	Text  string
	lines []int
}

// NewSource creates a source map for diagnostic spans.
func NewSource(file string, data []byte) Source {
	text := string(data)
	lines := []int{0}
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, i+1)
		}
	}
	return Source{File: file, Text: text, lines: lines}
}

// Pos converts a byte offset to a 1-based line and column.
func (s Source) Pos(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(s.Text) {
		offset = len(s.Text)
	}
	line := 0
	lo, hi := 0, len(s.lines)
	for lo < hi {
		mid := (lo + hi) / 2
		if s.lines[mid] <= offset {
			line = mid
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return Position{Line: line + 1, Column: offset - s.lines[line] + 1, Offset: offset}
}

// Span converts byte offsets to a source span.
func (s Source) Span(start, end int) Span {
	if end < start {
		end = start
	}
	return Span{File: s.File, Start: s.Pos(start), End: s.Pos(end)}
}
