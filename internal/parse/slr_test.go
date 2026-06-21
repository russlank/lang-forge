package parse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russlank/lang-forge/internal/diagnostics"
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
	span := diagnostics.Span{File: "ambig.lf", Start: diagnostics.Position{Line: 6, Column: 5}, End: diagnostics.Position{Line: 6, Column: 14}}
	g, diags := FromSpec(spec.Spec{
		Tokens: []spec.TokenDecl{{Name: "A"}},
		Grammar: spec.GrammarSpec{Start: "S", Rules: []spec.RuleSpec{
			{Name: "S", Alternatives: []spec.Alternative{{Symbols: []string{"S", "S"}, Span: span}, {Symbols: []string{"A"}}}},
		}},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	table := BuildSLR(g)
	if len(table.Conflicts) == 0 {
		t.Fatal("expected conflict")
	}
	conflict := table.Conflicts[0]
	if conflict.Hint == "" {
		t.Fatalf("conflict missing hint: %#v", conflict)
	}
	if len(conflict.ItemDetails) == 0 {
		t.Fatalf("conflict missing item details: %#v", conflict)
	}
	if conflict.ExistingRule == nil && conflict.IncomingRule == nil {
		t.Fatalf("conflict missing reduce rule details: %#v", conflict)
	}
	foundSpan := false
	foundDisplay := false
	for _, item := range conflict.ItemDetails {
		if item.Span.File == "ambig.lf" {
			foundSpan = true
		}
		if item.Display == "S -> S S •" {
			foundDisplay = true
		}
	}
	if !foundSpan {
		t.Fatalf("conflict item details did not preserve source span: %#v", conflict.ItemDetails)
	}
	if !foundDisplay {
		t.Fatalf("conflict item details did not include completed production display: %#v", conflict.ItemDetails)
	}
}

func TestBuild_UCDTAmbigYReportsSourceRichConflict(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "ucdt", "metas", "ambig.y")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, specDiags := spec.ParseYacc(data, path)
	if specDiags.HasErrors() {
		t.Fatalf("unexpected spec diagnostics: %v", specDiags)
	}
	g, grammarDiags := FromSpec(*parsed)
	if grammarDiags.HasErrors() {
		t.Fatalf("unexpected grammar diagnostics: %v", grammarDiags)
	}
	table := Build(g, parsed.Grammar.Algorithm)
	if len(table.Conflicts) == 0 {
		t.Fatal("expected conflict")
	}
	conflict := table.Conflicts[0]
	foundFixtureSpan := false
	foundCompletedExpr := false
	for _, item := range conflict.ItemDetails {
		if item.Span.File == path {
			foundFixtureSpan = true
		}
		if item.Display == "EXPR -> EXPR plus EXPR •" {
			foundCompletedExpr = true
		}
	}
	if !foundFixtureSpan {
		t.Fatalf("conflict did not preserve UCDT fixture source span: %#v", conflict.ItemDetails)
	}
	if !foundCompletedExpr {
		t.Fatalf("conflict did not include completed ambiguous expression item: %#v", conflict.ItemDetails)
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
