package parse

import (
	"testing"

	"github.com/russlank/lang-forge/internal/spec"
)

func TestNullableFirstFollow_WithEmptyProduction(t *testing.T) {
	g, diags := FromSpec(spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "A"}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"Opt", "A"}}}},
			{Name: "Opt", Alternatives: []spec.Alternative{{Symbols: nil}, {Symbols: []string{"A"}}}},
		}},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !g.Nullable()["Opt"] {
		t.Fatal("Opt should be nullable")
	}
	if !g.FirstSets()["Opt"]["A"] {
		t.Fatal("FIRST(Opt) should contain A")
	}
	if !g.FollowSets()["Opt"]["A"] {
		t.Fatal("FOLLOW(Opt) should contain A")
	}
}

func TestFromSpec_RejectsUndefinedSymbol(t *testing.T) {
	_, diags := FromSpec(spec.Spec{
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"Missing"}}}},
		}},
	})
	if !diags.HasErrors() {
		t.Fatal("expected undefined symbol diagnostic")
	}
}

func TestFromSpec_RejectsTokenNonterminalCollision(t *testing.T) {
	_, diags := FromSpec(spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "S"}, {Name: "A"}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"A"}}}},
		}},
	})
	if !diags.HasErrors() {
		t.Fatal("expected symbol collision diagnostic")
	}
}

func TestFromSpec_PreservesAlternativeActionsOnNormalizedRules(t *testing.T) {
	g, diags := FromSpec(spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "A"}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{
				{Symbols: []string{"A"}, Labels: []string{"value"}, Actions: map[string]string{"go": "make-s"}},
			}},
		}},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := g.Rules[0].Actions["go"]; got != "make-s" {
		t.Fatalf("action = %q, want make-s", got)
	}
	if got := g.Rules[0].Labels; len(got) != 1 || got[0] != "value" {
		t.Fatalf("labels = %#v, want value", got)
	}
}

func TestFromSpec_ValidatesSemanticTypes(t *testing.T) {
	_, terminalDiags := FromSpec(spec.Spec{
		Tokens:    []spec.TokenDecl{{Name: "A"}},
		Semantics: spec.SemanticSpec{Types: []spec.SemanticType{{Target: "go", Symbol: "A", Type: "string"}}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"A"}}}},
		}},
	})
	if !terminalDiags.HasErrors() {
		t.Fatal("expected terminal semantic type diagnostic")
	}

	_, missingDiags := FromSpec(spec.Spec{
		Semantics: spec.SemanticSpec{Types: []spec.SemanticType{{Target: "go", Symbol: "Missing", Type: "string"}}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{}}},
		}},
	})
	if !missingDiags.HasErrors() {
		t.Fatal("expected undefined semantic type diagnostic")
	}

	_, duplicateDiags := FromSpec(spec.Spec{
		Semantics: spec.SemanticSpec{Types: []spec.SemanticType{
			{Target: "go", Symbol: "S", Type: "string"},
			{Target: "go", Symbol: "S", Type: "int"},
		}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{}}},
		}},
	})
	if !duplicateDiags.HasErrors() {
		t.Fatal("expected duplicate semantic type diagnostic")
	}
}
