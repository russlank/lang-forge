package lex

import "testing"

func TestParseRegex_ModernAndLegacyRanges(t *testing.T) {
	modern, err := ParseRegex("[0-9]")
	if err != nil {
		t.Fatal(err)
	}
	if !modern.Set.Contains('5') || modern.Set.Contains(byteRune(5)) {
		t.Fatalf("[0-9] should mean digit characters, got %s", modern.Set)
	}
	legacy, err := ParseRegex("[1-32]")
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.Set.Contains('\n') || legacy.Set.Contains('A') {
		t.Fatalf("[1-32] should mean control code range, got %s", legacy.Set)
	}
	escapedLegacy, err := ParseRegex("[\\0-\\9]")
	if err != nil {
		t.Fatal(err)
	}
	if !escapedLegacy.Set.Contains('7') || escapedLegacy.Set.Contains('A') {
		t.Fatalf("[\\0-\\9] should mean digit characters, got %s", escapedLegacy.Set)
	}
}

func TestParseRegex_LegacySingleDigitByteRangeToEscapedPunctuation(t *testing.T) {
	expr, err := ParseRegex(`[1-\"]`)
	if err != nil {
		t.Fatal(err)
	}
	if expr.Kind != ExprSet {
		t.Fatalf("kind = %s", expr.Kind)
	}
	if !expr.Set.Contains(byteRune(1)) || !expr.Set.Contains('"') || expr.Set.Contains('#') {
		t.Fatalf("set = %s, want byte range 1 through quote", expr.Set)
	}
}

func TestExpandRefs_DetectsUndefinedAndRecursive(t *testing.T) {
	expr, err := ParseRegex("A")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExpandRefs(expr, map[string]*Expr{}); err == nil {
		t.Fatal("expected undefined reference error")
	}
	a := Ref("B")
	b := Ref("A")
	if _, err := ExpandRefs(a, map[string]*Expr{"A": a, "B": b}); err == nil {
		t.Fatal("expected recursive reference error")
	}
}

func TestParseRegex_UnicodeEscapesAndProperties(t *testing.T) {
	braced, err := ParseRegex(`"\u{1F600}"`)
	if err != nil {
		t.Fatal(err)
	}
	if !braced.Set.Contains('😀') {
		t.Fatalf("braced escape set = %s", braced.Set)
	}

	fixed, err := ParseRegex(`"\U0001F600"`)
	if err != nil {
		t.Fatal(err)
	}
	if !fixed.Set.Contains('😀') {
		t.Fatalf("fixed escape set = %s", fixed.Set)
	}

	letters, err := ParseRegex(`\p{L}+`)
	if err != nil {
		t.Fatal(err)
	}
	if letters.Kind != ExprPlus || !letters.Child.Set.Contains('å') || !letters.Child.Set.Contains('β') || letters.Child.Set.Contains('7') {
		t.Fatalf("letter property expression = %s", letters)
	}

	notLetters, err := ParseRegex(`[\P{Letter}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !notLetters.Set.Contains('7') || notLetters.Set.Contains('å') {
		t.Fatalf("negated letter property set = %s", notLetters.Set)
	}
}

func TestParseRegex_RejectsInvalidUnicodeScalars(t *testing.T) {
	for _, input := range []string{`"\uD800"`, `"\U00110000"`, `"\u{}"`} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseRegex(input); err == nil {
				t.Fatal("expected invalid Unicode scalar diagnostic")
			}
		})
	}
}

func TestParseRegex_RejectsPropertyRangeEndpoint(t *testing.T) {
	for _, input := range []string{`[\p{L}-z]`, `[a-\p{L}]`} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseRegex(input); err == nil {
				t.Fatal("expected property range endpoint diagnostic")
			}
		})
	}
}

func TestParseRegex_DotAndNegatedClassUseScannerDomain(t *testing.T) {
	dot, err := ParseRegex(".")
	if err != nil {
		t.Fatal(err)
	}
	if !dot.Set.Contains('å') || !dot.Set.Contains('😀') {
		t.Fatalf("dot set = %s", dot.Set)
	}
	if dot.Set.Contains(rune(0xD800)) {
		t.Fatalf("dot set includes surrogate: %s", dot.Set)
	}

	notQuote, err := ParseRegex(`[^"]+`)
	if err != nil {
		t.Fatal(err)
	}
	if !notQuote.Child.Set.Contains('β') || notQuote.Child.Set.Contains('"') {
		t.Fatalf("negated set = %s", notQuote.Child.Set)
	}
}

func TestExprNullable(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty literal", input: `""`, want: true},
		{name: "optional", input: `"a"?`, want: true},
		{name: "star", input: `"a"*`, want: true},
		{name: "plus nonempty", input: `"a"+`, want: false},
		{name: "alt", input: `""|"a"`, want: true},
		{name: "concat", input: `"" "a"`, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := ParseRegex(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got := expr.Nullable(); got != tc.want {
				t.Fatalf("Nullable(%s) = %t, want %t", tc.input, got, tc.want)
			}
		})
	}
}

func byteRune(b byte) rune {
	return rune(b)
}
