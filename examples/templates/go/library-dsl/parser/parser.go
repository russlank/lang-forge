//go:build langforge_generated

package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	librarydslgenerated "github.com/russlank/lang-forge/examples/templates/go/library-dsl/generated"
	"github.com/russlank/lang-forge/examples/templates/go/library-dsl/model"
)

// Parser is the reusable facade applications should depend on.
//
// The facade owns reducer wiring and exposes model.Document rather than the
// generated parser stack value type.
type Parser struct {
	reducers librarydslgenerated.ReducerMap
}

// New creates a parser facade with complete reducer coverage.
func New() Parser {
	return Parser{reducers: sharedReducers}
}

// Parse consumes source through the generated scanner lexeme source.
func (p Parser) Parse(source string) (model.Document, error) {
	value, err := librarydslgenerated.ParseWithReducerFromLexemeSource(librarydslgenerated.NewScanner(source), p.reducers)
	if err != nil {
		return model.Document{}, err
	}
	document, ok := value.(model.Document)
	if !ok {
		return model.Document{}, fmt.Errorf("parser final value has type %T, want model.Document", value)
	}
	return document, nil
}

// ParseTokens keeps the collection-based parser path available for tests and
// token-inspection tools. Production code should prefer Parse.
func (p Parser) ParseTokens(tokens []librarydslgenerated.Lexeme) (model.Document, error) {
	value, err := librarydslgenerated.ParseWithReducer(tokens, p.reducers)
	if err != nil {
		return model.Document{}, err
	}
	document, ok := value.(model.Document)
	if !ok {
		return model.Document{}, fmt.Errorf("parser final value has type %T, want model.Document", value)
	}
	return document, nil
}

// FormatError turns generated scanner/parser/reducer failures into a concise
// message for command-line demos. Larger applications can inspect generated
// ParseError diagnostics directly.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	var parseErr *librarydslgenerated.ParseError
	if errors.As(err, &parseErr) && len(parseErr.Diagnostics) > 0 {
		diagnostic := parseErr.Diagnostics[0]
		expected := make([]string, len(diagnostic.Expected))
		for i, token := range diagnostic.Expected {
			expected[i] = token.Display
		}
		return fmt.Sprintf("syntax error at %d:%d: unexpected %s; expected %s",
			diagnostic.StartLine,
			diagnostic.StartColumn,
			diagnostic.UnexpectedDisplay,
			strings.Join(expected, ", "))
	}
	return err.Error()
}

var sharedReducers = librarydslgenerated.ReducerMap{
	// Document : entries=Entries {go: document}
	librarydslgenerated.SemanticActionDocument: librarydslgenerated.TypedDocument(func(ctx librarydslgenerated.DocumentReduction) (model.Document, error) {
		return model.Document{Entries: ctx.Entries}, nil
	}),
	// Entries : head=Entry tail=EntriesTail {go: entries}
	librarydslgenerated.SemanticActionEntries: librarydslgenerated.TypedEntries(func(ctx librarydslgenerated.EntriesReduction) ([]model.Entry, error) {
		return prepend(ctx.Head, ctx.Tail), nil
	}),
	// Entries : %empty {go: entries.empty}
	librarydslgenerated.SemanticActionEntriesEmpty: librarydslgenerated.TypedEntriesEmpty(func(librarydslgenerated.EntriesEmptyReduction) ([]model.Entry, error) {
		return nil, nil
	}),
	// EntriesTail : head=Entry tail=EntriesTail {go: entries.tail.more}
	librarydslgenerated.SemanticActionEntriesTailMore: librarydslgenerated.TypedEntriesTailMore(func(ctx librarydslgenerated.EntriesTailMoreReduction) ([]model.Entry, error) {
		return prepend(ctx.Head, ctx.Tail), nil
	}),
	// EntriesTail : %empty {go: entries.tail.empty}
	librarydslgenerated.SemanticActionEntriesTailEmpty: librarydslgenerated.TypedEntriesTailEmpty(func(librarydslgenerated.EntriesTailEmptyReduction) ([]model.Entry, error) {
		return nil, nil
	}),
	// Entry : Set name=Ident Assign value=Value Semi {go: entry.set}
	librarydslgenerated.SemanticActionEntrySet: librarydslgenerated.TypedEntrySet(func(ctx librarydslgenerated.EntrySetReduction) (model.Entry, error) {
		return model.Entry{Kind: model.SetEntry, Name: ctx.Name.Text, Value: ctx.Value}, nil
	}),
	// Entry : Enable name=Ident Semi {go: entry.enable}
	librarydslgenerated.SemanticActionEntryEnable: librarydslgenerated.TypedEntryEnable(func(ctx librarydslgenerated.EntryEnableReduction) (model.Entry, error) {
		return model.Entry{Kind: model.EnableEntry, Name: ctx.Name.Text, Value: model.Value{Kind: model.BoolValue, Bool: true}}, nil
	}),
	// Value : token=Number {go: value.number}
	librarydslgenerated.SemanticActionValueNumber: librarydslgenerated.TypedValueNumber(reduceNumber),
	// Value : token=String {go: value.string}
	librarydslgenerated.SemanticActionValueString: librarydslgenerated.TypedValueString(func(ctx librarydslgenerated.ValueStringReduction) (model.Value, error) {
		text, err := unquote(ctx.Token.Text)
		if err != nil {
			return model.Value{}, fmt.Errorf("rule %d action %s label token: %w", ctx.Reduction.Rule, ctx.Reduction.Action, err)
		}
		return model.Value{Kind: model.StringValue, Text: text}, nil
	}),
	// Value : token=Ident {go: value.ident}
	librarydslgenerated.SemanticActionValueIdent: librarydslgenerated.TypedValueIdent(func(ctx librarydslgenerated.ValueIdentReduction) (model.Value, error) {
		return model.Value{Kind: model.IdentValue, Text: ctx.Token.Text}, nil
	}),
}

func reduceNumber(ctx librarydslgenerated.ValueNumberReduction) (model.Value, error) {
	value, err := strconv.Atoi(ctx.Token.Text)
	if err != nil {
		return model.Value{}, fmt.Errorf("rule %d action %s label token value %q is not a valid int: %w", ctx.Reduction.Rule, ctx.Reduction.Action, ctx.Token.Text, err)
	}
	return model.Value{Kind: model.NumberValue, Number: value}, nil
}

func prepend(head model.Entry, tail []model.Entry) []model.Entry {
	out := []model.Entry{head}
	return append(out, tail...)
}

func unquote(text string) (string, error) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", fmt.Errorf("string literal %q is not quoted", text)
	}
	body := text[1 : len(text)-1]
	var out strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] == '\\' {
			i++
			if i >= len(body) {
				return "", fmt.Errorf("string literal %q ends with an escape", text)
			}
		}
		out.WriteByte(body[i])
	}
	return out.String(), nil
}
