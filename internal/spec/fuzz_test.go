package spec

import "testing"

func FuzzParseCombinedSmoke(f *testing.F) {
	for _, seed := range []string{
		`%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A ;
`,
		`%scanner encoding=utf8 invalid=error
%target go
%package smoke
%token Number Plus
%start Expr
%% lexer
DIGIT = [0-9];
Number => token(Number);
"+" => token(Plus);
[1-32]+ => skip;
%% parser
Expr : Expr Plus Expr {go: add} | Number {go: number} ;
`,
		`%% lexer
%% parser
`,
		`%token 123Bad
%% lexer
%% parser
`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 8192 {
			t.Skip("limit smoke fuzz input size")
		}
		ParseCombined([]byte(input), "fuzz.lf")
	})
}
