package action

import (
	"testing"

	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
)

func TestBuild_GroupsRulesAndReportsTypedContext(t *testing.T) {
	grammar := &parse.Grammar{
		Terminals: map[string]bool{"Number": true, "Plus": true},
		Rules: []parse.Rule{
			{
				ID:      1,
				LHS:     "Expr",
				RHS:     []string{"Expr", "Plus", "Term"},
				Labels:  []string{"left", "", "right"},
				Actions: map[string]string{"go": "add"},
			},
			{
				ID:      2,
				LHS:     "Expr",
				RHS:     []string{"Term"},
				Labels:  []string{"value"},
				Actions: map[string]string{"go": "pass"},
			},
		},
	}
	semantics := spec.SemanticSpec{Types: []spec.SemanticType{
		{Target: "go", Symbol: "Expr", Type: "float64"},
		{Target: "go", Symbol: "Term", Type: "float64"},
	}}

	manifest := Build(grammar, semantics, "go")
	if len(manifest.Actions) != 2 {
		t.Fatalf("actions = %#v", manifest.Actions)
	}
	add := manifest.Actions[0]
	if add.ID != 1 || add.Name != "add" || !add.Typed || !add.ConsistentContext || add.ReturnType != "float64" {
		t.Fatalf("add action = %#v", add)
	}
	if got := add.Rules[0].RHS[0]; got.Label != "left" || got.Type != "float64" || got.Position != 1 {
		t.Fatalf("left operand = %#v", got)
	}
	if got := add.Rules[0].RHS[1]; got.Label != "" || got.Type != "Lexeme" {
		t.Fatalf("operator operand = %#v", got)
	}
}

func TestBuild_RejectsInconsistentSharedActionContext(t *testing.T) {
	grammar := &parse.Grammar{
		Terminals: map[string]bool{"Number": true},
		Rules: []parse.Rule{
			{ID: 1, LHS: "Expr", RHS: []string{"Expr"}, Labels: []string{"value"}, Actions: map[string]string{"go": "pass"}},
			{ID: 2, LHS: "Expr", RHS: []string{"Number"}, Labels: []string{"token"}, Actions: map[string]string{"go": "pass"}},
		},
	}
	semantics := spec.SemanticSpec{Types: []spec.SemanticType{
		{Target: "go", Symbol: "Expr", Type: "float64"},
	}}

	action := Build(grammar, semantics, "go").Actions[0]
	if action.Typed || action.ConsistentContext || action.TypeIssue == "" {
		t.Fatalf("action = %#v", action)
	}
}

func TestBuild_ReportsMissingResultType(t *testing.T) {
	grammar := &parse.Grammar{
		Terminals: map[string]bool{"Number": true},
		Rules: []parse.Rule{
			{ID: 1, LHS: "Expr", RHS: []string{"Number"}, Actions: map[string]string{"go": "number"}},
		},
	}
	action := Build(grammar, spec.SemanticSpec{}, "go").Actions[0]
	if action.Typed || action.TypeIssue == "" {
		t.Fatalf("action = %#v", action)
	}
}

func TestBuild_AllowsTypedContextWithoutLabeledOperands(t *testing.T) {
	grammar := &parse.Grammar{
		Terminals: map[string]bool{"Semi": true},
		Rules: []parse.Rule{
			{ID: 1, LHS: "Tail", RHS: []string{"Semi"}, Actions: map[string]string{"go": "tail.empty"}},
			{ID: 2, LHS: "Tail", Actions: map[string]string{"go": "tail.empty"}},
		},
	}
	semantics := spec.SemanticSpec{Types: []spec.SemanticType{
		{Target: "go", Symbol: "Tail", Type: "[]Item"},
	}}

	action := Build(grammar, semantics, "go").Actions[0]
	if !action.Typed || !action.ConsistentContext || action.TypeIssue != "" {
		t.Fatalf("action = %#v", action)
	}
}
