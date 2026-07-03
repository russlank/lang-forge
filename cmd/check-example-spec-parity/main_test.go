package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeSpecIgnoresTargetSpecificDirectivesAndFormatting(t *testing.T) {
	goSpec := `%target go
%package calc
%semantic go mode reducer
%start S
%token Number Plus

%% lexer
DIGIT = [0-9];
NUMBER = DIGIT+ ("." DIGIT+)?;
"+" => token(Plus);

%% parser
S : Number Plus Number {go: add} ;
`
	csharpSpec := `%package Demo.Generated
%target csharp
%semantic csharp mode reducer
%token Number Plus
%start S

%% lexer
DIGIT=[0-9];
NUMBER=DIGIT+("."DIGIT+)?;
"+"=>token(Plus);

%% parser
S:Number Plus Number
  {csharp: add}
  ;
`

	left, err := normalizeSpec(goSpec)
	if err != nil {
		t.Fatal(err)
	}
	right, err := normalizeSpec(csharpSpec)
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("normalized specs differ\nleft:\n%s\nright:\n%s", left, right)
	}
}

func TestNormalizeSpecPreservesWhitespaceInsideLiteralsAndClasses(t *testing.T) {
	spec := `%% lexer
WORD = [ A-Z]+;
"a b" => token(Word);
QUOTE = \" ( [1-127] )* \";
`
	normalized, err := normalizeSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(normalized, `[ A-Z]`) {
		t.Fatalf("character class whitespace was not preserved: %s", normalized)
	}
	if !strings.Contains(normalized, `"a b"`) {
		t.Fatalf("literal whitespace was not preserved: %s", normalized)
	}
	if !strings.Contains(normalized, `\"`) {
		t.Fatalf("escaped quote regex punctuation was not preserved: %s", normalized)
	}
}

func TestNormalizeSpecRejectsMalformedLiterals(t *testing.T) {
	if _, err := normalizeSpec(`%% lexer
"unterminated => token(Text);
`); err == nil {
		t.Fatal("expected unterminated literal error")
	}

	if _, err := normalizeSpec(`%% lexer
[A-Z => token(Text);
`); err == nil {
		t.Fatal("expected unterminated class error")
	}
}

func TestCheckFamilyReportsAllowlistedDifferences(t *testing.T) {
	dir := t.TempDir()
	baseline := filepath.Join(dir, "baseline.lf")
	variant := filepath.Join(dir, "variant.lf")
	if err := os.WriteFile(baseline, []byte(`%% lexer
"a" => token(A);
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variant, []byte(`%% lexer
"b" => token(B);
`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := checkFamily(exampleFamily{
		Name:  "test",
		Specs: []string{baseline, variant},
		AllowedDifferences: []allowedDifference{
			{
				Baseline: baseline,
				Path:     variant,
				Reason:   "variant intentionally uses another token spelling",
			},
		},
	})
	if err != nil {
		t.Fatalf("allowlisted mismatch failed: %v", err)
	}
}

func TestNormalizeSpecPreservesNamedRHSLabels(t *testing.T) {
	normalized, err := normalizeSpec(`%semantic go type Expr float64
%% parser
Expr : left=Expr Plus right=Term {go: add} ;
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(normalized, "left=Expr") || !strings.Contains(normalized, "right=Term") {
		t.Fatalf("normalized spec lost RHS labels: %s", normalized)
	}
}

func TestNormalizeSpecPreservesPortableActionLabelText(t *testing.T) {
	goSpec, err := normalizeSpec(`%% parser
S : value=Expr {go: program.withParameters} ;
`)
	if err != nil {
		t.Fatal(err)
	}
	cSpec, err := normalizeSpec(`%% parser
S : value=Expr {c: program.withParameters} ;
`)
	if err != nil {
		t.Fatal(err)
	}
	if goSpec != cSpec {
		t.Fatalf("normalized action labels differ\nleft:\n%s\nright:\n%s", goSpec, cSpec)
	}
	if !strings.Contains(goSpec, "{ACTION:program.withParameters}") || strings.Contains(goSpec, "{go:") {
		t.Fatalf("action target was not normalized while preserving label: %s", goSpec)
	}
}

func TestNormalizeSpecDetectsActionLabelDrift(t *testing.T) {
	goSpec, err := normalizeSpec(`%% parser
S : value=Expr {go: program.withParameters} ;
`)
	if err != nil {
		t.Fatal(err)
	}
	cSpec, err := normalizeSpec(`%% parser
S : value=Expr {c: program.with_parameters} ;
`)
	if err != nil {
		t.Fatal(err)
	}
	if goSpec == cSpec {
		t.Fatalf("normalized spec should preserve action label drift:\n%s", goSpec)
	}
}
