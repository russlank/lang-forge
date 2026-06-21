package lex

import "testing"

func FuzzParseRegexSmoke(f *testing.F) {
	for _, seed := range []string{
		`"a"`,
		`[0-9]+`,
		`DIGIT+ ("." DIGIT+)?`,
		`\p{L}+`,
		`[^"]*`,
		`"\u{1F600}"`,
		`("(" | ")")*`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 1024 {
			t.Skip("limit smoke fuzz input size")
		}
		expr, err := ParseRegex(input)
		if err != nil {
			return
		}
		_ = expr.Nullable()
	})
}
