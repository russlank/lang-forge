package golang

import (
	"strings"
	"testing"

	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
)

func TestGoPackageName_ExplicitPackageMustBeValid(t *testing.T) {
	for _, name := range []string{"bad-name", "type", "_", "123abc"} {
		if got, err := goPackageName(name, "fallback"); err == nil {
			t.Fatalf("goPackageName(%q) = %q, nil error; want error", name, got)
		}
	}
	if got, err := goPackageName("calc", "fallback"); err != nil || got != "calc" {
		t.Fatalf("goPackageName explicit calc = %q, %v; want calc, nil", got, err)
	}
}

func TestValidateSemanticTypesRejectsInvalidGoType(t *testing.T) {
	err := validateSemanticTypes(spec.SemanticSpec{Types: []spec.SemanticType{
		{Target: "go", Symbol: "Expr", Type: "[]"},
	}}, "go")
	if err == nil || !strings.Contains(err.Error(), "Expr") {
		t.Fatalf("error = %v", err)
	}
}

func TestGoPackageName_DefaultSanitizesOutputDirectory(t *testing.T) {
	cases := map[string]string{
		"calc-generated": "calc_generated",
		"123calc":        "calc",
		"type":           "langforge_type",
	}
	for dir, want := range cases {
		if got, err := goPackageName("", dir); err != nil || got != want {
			t.Fatalf("goPackageName default %q = %q, %v; want %q, nil", dir, got, err, want)
		}
	}
}

func TestSemanticActions_GeneratesStableGoConstants(t *testing.T) {
	rules := []parseRuleForTest{
		{action: "add"},
		{action: "program.withParameters"},
		{action: "program-withParameters"},
		{action: "add"},
	}
	actions := semanticActionsForTest(rules)
	if len(actions) != 3 {
		t.Fatalf("actions len = %d, want 3", len(actions))
	}
	want := []SemanticAction{
		{ID: 1, Name: "add", Constant: "SemanticActionAdd"},
		{ID: 2, Name: "program.withParameters", Constant: "SemanticActionProgramWithParameters"},
		{ID: 3, Name: "program-withParameters", Constant: "SemanticActionProgramWithParameters2"},
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("actions[%d] = %#v, want %#v", i, actions[i], want[i])
		}
	}
}

type parseRuleForTest struct {
	action string
}

func semanticActionsForTest(rules []parseRuleForTest) []SemanticAction {
	converted := make([]parse.Rule, 0, len(rules))
	for i, rule := range rules {
		converted = append(converted, parse.Rule{
			ID:      i + 1,
			Actions: map[string]string{"go": rule.action},
		})
	}
	return semanticActions(converted, "go")
}
