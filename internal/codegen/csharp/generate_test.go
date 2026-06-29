package csharp

import (
	"strings"
	"testing"

	"github.com/russlank/lang-forge/internal/action"
	"github.com/russlank/lang-forge/internal/parse"
)

func TestCSharpNamespace_ExplicitNamespaceMustBeValid(t *testing.T) {
	for _, name := range []string{"bad-name", "123.Bad", "Good..Bad"} {
		if got, err := csharpNamespace(name, "fallback"); err == nil {
			t.Fatalf("csharpNamespace(%q) = %q, nil error; want error", name, got)
		}
	}
	if got, err := csharpNamespace("LangForge.Examples.Calc.Generated", "fallback"); err != nil || got != "LangForge.Examples.Calc.Generated" {
		t.Fatalf("csharpNamespace explicit = %q, %v", got, err)
	}
}

func TestCSharpNamespace_DefaultSanitizesOutputDirectory(t *testing.T) {
	if got, err := csharpNamespace("", "calc-generated"); err != nil || got != "LangForge.Generated.CalcGenerated" {
		t.Fatalf("csharpNamespace default = %q, %v", got, err)
	}
}

func TestSemanticActions_GeneratesStableCSharpConstants(t *testing.T) {
	rules := []parse.Rule{
		{ID: 1, Actions: map[string]string{"csharp": "add"}},
		{ID: 2, Actions: map[string]string{"csharp": "program.withParameters"}},
		{ID: 3, Actions: map[string]string{"csharp": "program-withParameters"}},
		{ID: 4, Actions: map[string]string{"csharp": "add"}},
	}
	actions := semanticActions(rules, "csharp")
	want := []SemanticAction{
		{ID: 1, Name: "add", Constant: "Add"},
		{ID: 2, Name: "program.withParameters", Constant: "ProgramWithParameters"},
		{ID: 3, Name: "program-withParameters", Constant: "ProgramWithParameters2"},
	}
	if len(actions) != len(want) {
		t.Fatalf("actions len = %d, want %d", len(actions), len(want))
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("actions[%d] = %#v, want %#v", i, actions[i], want[i])
		}
	}
}

func TestRenderTypedReductionContexts_GeneratesCSharpAdapters(t *testing.T) {
	manifest := action.Manifest{
		Target: "csharp",
		Actions: []action.Action{
			{
				ID:         1,
				Name:       "add",
				Typed:      true,
				ReturnType: "double",
				Rules: []action.Rule{
					{
						ID:         3,
						LHS:        "Expr",
						ReturnType: "double",
						Typed:      true,
						RHS: []action.Operand{
							{Position: 1, Symbol: "Expr", Label: "left", Type: "double"},
							{Position: 2, Symbol: "Plus", Type: "Lexeme"},
							{Position: 3, Symbol: "Term", Label: "right", Type: "double"},
						},
					},
				},
				ConsistentContext: true,
			},
		},
	}
	actions := []SemanticAction{{ID: 1, Name: "add", Constant: "Add"}}

	var b strings.Builder
	renderTypedReductionContexts(&b, manifest, actions)
	got := b.String()
	for _, want := range []string{
		"internal sealed record AddReduction(Reduction Reduction, double Left, double Right);",
		"internal delegate double AddHandler(AddReduction ctx);",
		"internal static class SemanticReducerContexts",
		"internal static AddReduction NewAddReduction(Reduction ctx)",
		"SemanticValueAs<double>(ctx, \"left\")",
		"internal static Func<Reduction, object?> TypedAdd(AddHandler handler)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("typed C# contexts missing %q in:\n%s", want, got)
		}
	}
}
