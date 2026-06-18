package spec

import "testing"

func TestParseCombined_DoesNotSplitAlternationInsideAction(t *testing.T) {
	input := []byte(`%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
%% parser
S : A {go: value := "a|b"} | B ;
`)
	parsed, diags := ParseCombined(input, "test.lf")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := len(parsed.Grammar.Rules[0].Alternatives); got != 2 {
		t.Fatalf("alternatives = %d, want 2", got)
	}
	if got := parsed.Grammar.Rules[0].Alternatives[0].Actions["go"]; got != `value := "a|b"` {
		t.Fatalf("action = %q", got)
	}
}

func TestParseCombined_RejectsBadTokenName(t *testing.T) {
	_, diags := ParseCombined([]byte("%token 123Bad\n%% lexer\n%% parser\n"), "bad.lf")
	if !diags.HasErrors() {
		t.Fatal("expected diagnostics for bad token")
	}
}

func TestParseCombined_ParserAlgorithmDirective(t *testing.T) {
	parsed, diags := ParseCombined([]byte(`%type ielr
%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A ;
`), "algorithm.lf")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if parsed.Grammar.Algorithm != "ielr" {
		t.Fatalf("algorithm = %q", parsed.Grammar.Algorithm)
	}

	_, diags = ParseCombined([]byte(`%type magic
%% lexer
%% parser
`), "bad-algorithm.lf")
	if !diags.HasErrors() {
		t.Fatal("expected diagnostic for unknown algorithm")
	}
}

func TestParseCombined_ScannerDirective(t *testing.T) {
	parsed, diags := ParseCombined([]byte(`%scanner encoding=utf8 invalid=error newline=lf
%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A ;
`), "scanner.lf")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if parsed.Scanner.Encoding != ScannerEncodingUTF8 || parsed.Scanner.Invalid != ScannerInvalidError || parsed.Scanner.Newline != "lf" {
		t.Fatalf("scanner = %#v", parsed.Scanner)
	}

	parsed, diags = ParseCombined([]byte(`%scanner utf8
%% lexer
%% parser
`), "short-scanner.lf")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics for short form: %v", diags)
	}
	if parsed.Scanner.Encoding != ScannerEncodingUTF8 {
		t.Fatalf("scanner encoding = %q", parsed.Scanner.Encoding)
	}

	_, diags = ParseCombined([]byte(`%scanner encoding=latin1
%% lexer
%% parser
`), "bad-scanner.lf")
	if !diags.HasErrors() {
		t.Fatal("expected diagnostic for unsupported scanner encoding")
	}
}

func TestParseCombined_SemanticDirectives(t *testing.T) {
	parsed, diags := ParseCombined([]byte(`%target go
%semantic go mode inline
%semantic go import sem "example.test/semantics"
%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A {go: return sem.Reduce(ctx)} ;
`), "semantic.lf")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := parsed.Semantics.ModeFor("go"); got != SemanticModeInline {
		t.Fatalf("mode = %q, want inline", got)
	}
	includes := parsed.Semantics.IncludesFor("go")
	if len(includes) != 1 {
		t.Fatalf("includes = %#v", includes)
	}
	if includes[0].Alias != "sem" || includes[0].Path != "example.test/semantics" {
		t.Fatalf("include = %#v", includes[0])
	}
	if got := parsed.Grammar.Rules[0].Alternatives[0].Actions["go"]; got != "return sem.Reduce(ctx)" {
		t.Fatalf("action = %q", got)
	}
}

func TestParseYacc_SemanticDirectives(t *testing.T) {
	parsed, diags := ParseYacc([]byte(`%semantic go mode reducer
%semantic go import drawsem "example.test/drawsem"
%token A
%start S
%%
S : A {go: makeS} ;
`), "semantic.y")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := parsed.Semantics.ModeFor("go"); got != SemanticModeReducer {
		t.Fatalf("mode = %q, want reducer", got)
	}
	includes := parsed.Semantics.IncludesFor("go")
	if len(includes) != 1 || includes[0].Alias != "drawsem" || includes[0].Path != "example.test/drawsem" {
		t.Fatalf("includes = %#v", includes)
	}
	if got := parsed.Grammar.Rules[0].Alternatives[0].Actions["go"]; got != "makeS" {
		t.Fatalf("action = %q", got)
	}
}

func TestParseLex_DoesNotSplitQuotedPercentSeparator(t *testing.T) {
	parsed, diags := ParseLex([]byte(`DIGIT = [\0-\9];
%%
"%%" : #{LEX_DPercent#}
DIGIT : #{LEX_Digit#}
%%
`), "legacy.l")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got, want := len(parsed.Lexer.Rules), 2; got != want {
		t.Fatalf("rules = %d, want %d", got, want)
	}
	if got := parsed.Lexer.Rules[0].Pattern; got != `"%%"` {
		t.Fatalf("first pattern = %q", got)
	}
}

func TestParseLex_PreservesQuotedBlockCommentDelimiters(t *testing.T) {
	parsed, diags := ParseLex([]byte(`COMMENT = "/*" [1-255]* "*/";
%%
COMMENT : #{LEX_Comment#}
%%
`), "legacy.l")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got, want := parsed.Lexer.Definitions[0].Pattern, `"/*" [1-255]* "*/"`; got != want {
		t.Fatalf("pattern = %q, want %q", got, want)
	}
}

func TestParseLex_PreservesCommentDelimitersInClass(t *testing.T) {
	parsed, diags := ParseLex([]byte(`DELIM = [/*];
%%
DELIM : #{YACC_Delim#}
%%
`), "legacy.l")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got, want := parsed.Lexer.Definitions[0].Pattern, `[/*]`; got != want {
		t.Fatalf("pattern = %q, want %q", got, want)
	}
}

func TestParseLex_DoesNotSplitPercentSeparatorInClass(t *testing.T) {
	parsed, diags := ParseLex([]byte(`PERCENT = [%%];
%%
PERCENT : #{YACC_Percent#}
%%
`), "legacy.l")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got, want := parsed.Lexer.Definitions[0].Pattern, `[%%]`; got != want {
		t.Fatalf("pattern = %q, want %q", got, want)
	}
	if got, want := len(parsed.Lexer.Rules), 1; got != want {
		t.Fatalf("rules = %d, want %d", got, want)
	}
}
