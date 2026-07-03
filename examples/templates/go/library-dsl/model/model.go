package model

import (
	"fmt"
	"strconv"
)

// Document is the stable AST root returned by the parser facade.
type Document struct {
	Entries []Entry
}

// EntryKind describes which grammar alternative produced an Entry.
type EntryKind string

const (
	// SetEntry comes from: Entry : Set name=Ident Assign value=Value Semi.
	SetEntry EntryKind = "set"
	// EnableEntry comes from: Entry : Enable name=Ident Semi.
	EnableEntry EntryKind = "enable"
)

// Entry is one top-level DSL statement.
type Entry struct {
	Kind  EntryKind
	Name  string
	Value Value
}

// ValueKind identifies which Value grammar alternative was reduced.
type ValueKind string

const (
	// NumberValue comes from: Value : token=Number.
	NumberValue ValueKind = "number"
	// StringValue comes from: Value : token=String.
	StringValue ValueKind = "string"
	// IdentValue comes from: Value : token=Ident.
	IdentValue ValueKind = "identifier"
	// BoolValue is used by enable statements, which have no explicit Value RHS.
	BoolValue ValueKind = "bool"
)

// Value is a compact semantic value for assignment and enable statements.
type Value struct {
	Kind   ValueKind
	Text   string
	Number int
	Bool   bool
}

// String returns a user-facing representation suitable for reports.
func (v Value) String() string {
	switch v.Kind {
	case NumberValue:
		return strconv.Itoa(v.Number)
	case StringValue:
		return strconv.Quote(v.Text)
	case IdentValue:
		return v.Text
	case BoolValue:
		return strconv.FormatBool(v.Bool)
	default:
		return fmt.Sprintf("<unknown:%s>", v.Kind)
	}
}

// Settings lowers the ordered AST into a map for applications that prefer
// lookup semantics over source order.
func (d Document) Settings() map[string]Value {
	out := make(map[string]Value, len(d.Entries))
	for _, entry := range d.Entries {
		out[entry.Name] = entry.Value
	}
	return out
}
