package parse

import (
	"testing"

	"github.com/russlank/lang-forge/internal/parseralgo"
	"github.com/russlank/lang-forge/internal/spec"
)

func TestBuildSLR_CalcGrammarHasNoConflicts(t *testing.T) {
	g, diags := FromSpec(calcSpec())
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	table := BuildSLR(g)
	if len(table.Conflicts) != 0 {
		t.Fatalf("conflicts = %#v", table.Conflicts)
	}
	if len(table.States) == 0 {
		t.Fatal("expected states")
	}
}

func TestBuildSLR_AmbiguousGrammarReportsConflict(t *testing.T) {
	g, diags := FromSpec(spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "A"}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"S", "S"}}, {Symbols: []string{"A"}}}},
		}},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	table := BuildSLR(g)
	if len(table.Conflicts) == 0 {
		t.Fatal("expected conflict")
	}
}

func TestBuildCanonicalLR1_GrammarThatSLRRejects(t *testing.T) {
	g, diags := FromSpec(lr1ButNotSLRSpec())
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	slr := BuildSLR(g)
	if len(slr.Conflicts) == 0 {
		t.Fatal("expected SLR conflict")
	}
	canonical := BuildCanonicalLR1(g)
	if len(canonical.Conflicts) != 0 {
		t.Fatalf("canonical conflicts = %#v", canonical.Conflicts)
	}
	if canonical.Algorithm != "canonical" {
		t.Fatalf("algorithm = %q", canonical.Algorithm)
	}
	if len(canonical.States) == 0 || len(canonical.States[0].LR1Items) == 0 {
		t.Fatal("expected canonical LR(1) items in table states")
	}
}

func TestBuildLALR_GrammarThatSLRRejects(t *testing.T) {
	g, diags := FromSpec(lr1ButNotSLRSpec())
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	lalr := BuildLALR(g)
	if len(lalr.Conflicts) != 0 {
		t.Fatalf("lalr conflicts = %#v", lalr.Conflicts)
	}
	if lalr.Algorithm != "lalr" {
		t.Fatalf("algorithm = %q", lalr.Algorithm)
	}
}

func TestBuildIELR_MatchesLALRWhenCoreMergesAreSafe(t *testing.T) {
	g, diags := FromSpec(lr1ButNotSLRSpec())
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	lalr := BuildLALR(g)
	ielr := BuildIELR(g)
	if len(ielr.Conflicts) != 0 {
		t.Fatalf("ielr conflicts = %#v", ielr.Conflicts)
	}
	if ielr.Algorithm != parseralgo.IELR {
		t.Fatalf("algorithm = %q", ielr.Algorithm)
	}
	if len(ielr.States) != len(lalr.States) {
		t.Fatalf("IELR states = %d, LALR states = %d; want same count for safe core merge", len(ielr.States), len(lalr.States))
	}
}

func TestBuildIELR_SplitsMysteriousLALRConflict(t *testing.T) {
	g, diags := FromSpec(mysteriousConflictSpec())
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	lalr := BuildLALR(g)
	if len(lalr.Conflicts) == 0 {
		t.Fatal("expected LALR conflict")
	}
	ielr := BuildIELR(g)
	if len(ielr.Conflicts) != 0 {
		t.Fatalf("ielr conflicts = %#v", ielr.Conflicts)
	}
	canonical := BuildCanonicalLR1(g)
	if len(canonical.Conflicts) != 0 {
		t.Fatalf("canonical conflicts = %#v", canonical.Conflicts)
	}
	if len(ielr.States) <= len(lalr.States) {
		t.Fatalf("IELR states = %d, LALR states = %d; want IELR to split the unsafe LALR merge", len(ielr.States), len(lalr.States))
	}
	if len(ielr.States) > len(canonical.States) {
		t.Fatalf("IELR states = %d, canonical states = %d; IELR should not exceed canonical LR(1)", len(ielr.States), len(canonical.States))
	}
}

func TestBuild_NormalizesAlgorithmToLALRDefault(t *testing.T) {
	g, diags := FromSpec(calcSpec())
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	for _, algorithm := range []string{"", "unknown"} {
		table := Build(g, algorithm)
		if table.Algorithm != parseralgo.LALR {
			t.Fatalf("Build(%q) algorithm = %q, want %q", algorithm, table.Algorithm, parseralgo.LALR)
		}
	}
}

func calcSpec() spec.Spec {
	return spec.Spec{
		Tokens: []spec.TokenDecl{
			{Name: "Number"}, {Name: "Plus"}, {Name: "Minus"}, {Name: "Mul"}, {Name: "Div"}, {Name: "LParen"}, {Name: "RParen"},
		},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"Expr"}}}},
			{Name: "Expr", Alternatives: []spec.Alternative{
				{Symbols: []string{"Expr", "Plus", "Term"}},
				{Symbols: []string{"Expr", "Minus", "Term"}},
				{Symbols: []string{"Term"}},
			}},
			{Name: "Term", Alternatives: []spec.Alternative{
				{Symbols: []string{"Term", "Mul", "Factor"}},
				{Symbols: []string{"Term", "Div", "Factor"}},
				{Symbols: []string{"Factor"}},
			}},
			{Name: "Factor", Alternatives: []spec.Alternative{
				{Symbols: []string{"Number"}},
				{Symbols: []string{"LParen", "Expr", "RParen"}},
				{Symbols: []string{"Minus", "Factor"}},
			}},
		}},
	}
}

func mysteriousConflictSpec() spec.Spec {
	return spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "ID"}, {Name: "Colon"}, {Name: "Comma"}},
		Grammar: spec.GrammarSpec{Start: "Def", Rules: []spec.RuleSpec{
			{Name: "Def", Alternatives: []spec.Alternative{{Symbols: []string{"ParamSpec", "ReturnSpec", "Comma"}}}},
			{Name: "ParamSpec", Alternatives: []spec.Alternative{
				{Symbols: []string{"Type"}},
				{Symbols: []string{"NameList", "Colon", "Type"}},
			}},
			{Name: "ReturnSpec", Alternatives: []spec.Alternative{
				{Symbols: []string{"Type"}},
				{Symbols: []string{"Name", "Colon", "Type"}},
			}},
			{Name: "Type", Alternatives: []spec.Alternative{{Symbols: []string{"ID"}}}},
			{Name: "Name", Alternatives: []spec.Alternative{{Symbols: []string{"ID"}}}},
			{Name: "NameList", Alternatives: []spec.Alternative{
				{Symbols: []string{"Name"}},
				{Symbols: []string{"Name", "Comma", "NameList"}},
			}},
		}},
	}
}

func lr1ButNotSLRSpec() spec.Spec {
	return spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "ID"}, {Name: "Star"}, {Name: "Eq"}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{
				{Symbols: []string{"L", "Eq", "R"}},
				{Symbols: []string{"R"}},
			}},
			{Name: "L", Alternatives: []spec.Alternative{
				{Symbols: []string{"Star", "R"}},
				{Symbols: []string{"ID"}},
			}},
			{Name: "R", Alternatives: []spec.Alternative{{Symbols: []string{"L"}}}},
		}},
	}
}
