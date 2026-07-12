package lex

import (
	"testing"

	"github.com/russlank/lang-forge/internal/spec"
)

func TestDFA_LongestMatchThenRulePriority(t *testing.T) {
	lexer := spec.LexerSpec{
		Rules: []spec.LexRule{
			{Pattern: `"if"`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "If"}},
			{Pattern: `[A-Za-z]+`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Ident"}},
			{Pattern: `[1-32]+`, Action: spec.LexAction{Kind: spec.ActionSkip}},
		},
	}
	dfa, diags := BuildFromSpec(lexer)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	tokens, err := dfa.Tokenize("if iffy", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 {
		t.Fatalf("tokens len = %d", len(tokens))
	}
	if tokens[0].Name != "If" || tokens[0].Lexeme != "if" {
		t.Fatalf("first token = %#v, want If", tokens[0])
	}
	if tokens[1].Name != "Ident" || tokens[1].Lexeme != "iffy" {
		t.Fatalf("second token = %#v, want Ident/iffy", tokens[1])
	}
}

func TestDFA_HiddenChannelCanBeIncluded(t *testing.T) {
	lexer := spec.LexerSpec{
		Rules: []spec.LexRule{
			{Pattern: `"a"`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "A"}},
			{Pattern: `[1-32]+`, Action: spec.LexAction{Kind: spec.ActionChannel, Channel: "Trivia"}},
		},
	}
	dfa, diags := BuildFromSpec(lexer)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	without, err := dfa.Tokenize("a a", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(without) != 2 {
		t.Fatalf("without hidden len = %d, want 2", len(without))
	}
	with, err := dfa.Tokenize("a a", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(with) != 3 || with[1].Channel != "Trivia" {
		t.Fatalf("with hidden = %#v", with)
	}
}

func TestDFA_ReportsUnmatchedInput(t *testing.T) {
	dfa, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `"a"`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "A"}},
	}})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if _, err := dfa.Tokenize("b", false); err == nil {
		t.Fatal("expected unmatched input error")
	}
}

func TestBuildFromSpec_AcceptsLegacyByteRanges(t *testing.T) {
	_, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `[128-255]`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "HighByte"}},
	}})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics for byte-oriented range: %v", diags)
	}
}

func TestBuildFromSpec_AcceptsUnicodeScalarRanges(t *testing.T) {
	dfa, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `[\u0100-\u0101]`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Wide"}},
	}})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics for Unicode scalar range: %v", diags)
	}
	tokens, err := dfa.Tokenize("Āā", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 || tokens[0].Lexeme != "Ā" || tokens[1].Start != len("Ā") {
		t.Fatalf("tokens = %#v", tokens)
	}
}

func TestBuildFromSpec_RejectsSurrogateRange(t *testing.T) {
	_, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `[\uD800-\uDFFF]`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Bad"}},
	}})
	if !diags.HasErrors() {
		t.Fatal("expected surrogate diagnostic")
	}
}

func TestBuildFromSpec_RejectsEmptyCharacterClass(t *testing.T) {
	_, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `[]`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Empty"}},
	}})
	if !diags.HasErrors() {
		t.Fatal("expected empty character class diagnostic")
	}
}

func TestBuildFromSpec_RejectsNullableRules(t *testing.T) {
	_, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `""`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Empty"}},
		{Pattern: `"a"*`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "ManyA"}},
		{Pattern: `"b"?`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "MaybeB"}},
	}})
	if !diags.HasErrors() {
		t.Fatal("expected nullable-rule diagnostics")
	}
	if len(diags) != 3 {
		t.Fatalf("diagnostics = %#v, want one per nullable rule", diags)
	}
}

func TestDFA_MinimizePreservesTokenizationAndMergesEquivalentStates(t *testing.T) {
	lexer := spec.LexerSpec{
		Rules: []spec.LexRule{
			{Pattern: `"ab"|"cb"`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Word"}},
			{Pattern: `[1-32]+`, Action: spec.LexAction{Kind: spec.ActionSkip}},
		},
	}
	dfa, diags := BuildFromSpec(lexer)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	tokens, err := dfa.Tokenize("ab cb", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 || tokens[0].Lexeme != "ab" || tokens[1].Lexeme != "cb" {
		t.Fatalf("tokens = %#v", tokens)
	}
	if len(dfa.States) >= 6 {
		t.Fatalf("expected minimization to merge equivalent suffix states, got %d states", len(dfa.States))
	}
}

func TestDFA_TokenizesUTF8WithByteSpansAndScalarPositions(t *testing.T) {
	lexer := spec.LexerSpec{
		Rules: []spec.LexRule{
			{Pattern: `\p{L}+`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Word"}},
			{Pattern: `[1-32]+`, Action: spec.LexAction{Kind: spec.ActionSkip}},
		},
	}
	dfa, diags := BuildFromSpec(lexer)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	tokens, err := dfa.Tokenize("åβ\ncat", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 {
		t.Fatalf("tokens = %#v", tokens)
	}
	if tokens[0].Lexeme != "åβ" || tokens[0].Start != 0 || tokens[0].End != len("åβ") {
		t.Fatalf("first token spans = %#v", tokens[0])
	}
	if tokens[0].StartLine != 1 || tokens[0].StartColumn != 1 || tokens[0].EndColumn != 3 {
		t.Fatalf("first token position = %#v", tokens[0])
	}
	if tokens[1].Lexeme != "cat" || tokens[1].StartLine != 2 || tokens[1].StartColumn != 1 {
		t.Fatalf("second token = %#v", tokens[1])
	}
}

func TestDFA_RejectsMalformedUTF8WithoutLooping(t *testing.T) {
	dfa, diags := BuildFromSpec(spec.LexerSpec{Rules: []spec.LexRule{
		{Pattern: `.`, Action: spec.LexAction{Kind: spec.ActionToken, Token: "Any"}},
	}})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if _, err := dfa.Tokenize(string([]byte{0xff}), false); err == nil {
		t.Fatal("expected invalid UTF-8 error")
	}
	tokens, err := dfa.Tokenize("a"+string([]byte{0xff}), false)
	if err == nil {
		t.Fatal("expected invalid UTF-8 error after first token")
	}
	if len(tokens) != 0 {
		t.Fatalf("Tokenize should not return partial tokens with an error: %#v", tokens)
	}
}
