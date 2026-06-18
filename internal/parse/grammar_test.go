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
				{Symbols: []string{"A"}, Actions: map[string]string{"go": "make-s"}},
			}},
		}},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := g.Rules[0].Actions["go"]; got != "make-s" {
		t.Fatalf("action = %q, want make-s", got)
	}
}
