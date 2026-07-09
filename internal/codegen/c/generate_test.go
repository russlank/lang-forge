package c

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
)

func TestGenerateWritesConventionalCFilesAndMetadata(t *testing.T) {
	parsed, diagnostics := spec.ParseCombined([]byte(`%target c
%package calc-demo
%semantic c mode reducer
%semantic c type S double
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : left=A right=B {c: program.withParameters}
  | left=B right=A {c: addObject}
  ;
`), "calc.lf")
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics: %v", diagnostics)
	}
	dfa, lexDiagnostics := lex.BuildFromSpecWithScanner(parsed.Lexer, parsed.Scanner)
	if lexDiagnostics.HasErrors() {
		t.Fatalf("lex diagnostics: %v", lexDiagnostics)
	}
	grammar, grammarDiagnostics := parse.FromSpec(*parsed)
	if grammarDiagnostics.HasErrors() {
		t.Fatalf("grammar diagnostics: %v", grammarDiagnostics)
	}
	table := parse.Build(grammar, parsed.Grammar.Algorithm)
	if len(table.Conflicts) != 0 {
		t.Fatalf("conflicts: %#v", table.Conflicts)
	}

	out := t.TempDir()
	if err := Generate(Input{Spec: parsed, DFA: dfa, Grammar: grammar, ParseTable: table}, out); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"langforge.actions.json",
		"langforge.manifest.json",
		"langforge.tables.json",
		"tokens.h",
		"scanner.h",
		"scanner.c",
		"parser.h",
		"parser_typed.h",
		"parser.c",
	} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	manifest := readGeneratedFile(t, out, "langforge.manifest.json")
	for _, fragment := range []string{`"target": "c"`, `"prefix": "calc_demo"`, `"CALC_DEMO_ACTION_PROGRAM_WITH_PARAMETERS"`, `"CALC_DEMO_ACTION_ADD_OBJECT"`} {
		if !strings.Contains(manifest, fragment) {
			t.Fatalf("manifest missing %q:\n%s", fragment, manifest)
		}
	}
	actionManifest := readGeneratedFile(t, out, "langforge.actions.json")
	for _, fragment := range []string{`"name": "program.withParameters"`, `"name": "addObject"`, `"lhs": "S"`, `"symbol": "A"`, `"label": "left"`, `"typed": true`} {
		if !strings.Contains(actionManifest, fragment) {
			t.Fatalf("action manifest missing %q:\n%s", fragment, actionManifest)
		}
	}

	tokens := readGeneratedFile(t, out, "tokens.h")
	for _, fragment := range []string{"typedef enum calc_demo_token", "CALC_DEMO_TOKEN_EOF", "CALC_DEMO_TOKEN_A"} {
		if !strings.Contains(tokens, fragment) {
			t.Fatalf("tokens.h missing %q:\n%s", fragment, tokens)
		}
	}

	parserHeader := readGeneratedFile(t, out, "parser.h")
	for _, fragment := range []string{"typedef enum calc_demo_semantic_action", "CALC_DEMO_ACTION_PROGRAM_WITH_PARAMETERS", "CALC_DEMO_ACTION_ADD_OBJECT", "const char **labels", "calc_demo_reduction_value_for", "calc_demo_parse_result", "calc_demo_parse_recovering", "calc_demo_parse_result_free"} {
		if !strings.Contains(parserHeader, fragment) {
			t.Fatalf("parser.h missing %q:\n%s", fragment, parserHeader)
		}
	}

	typedHeader := readGeneratedFile(t, out, "parser_typed.h")
	for _, fragment := range []string{"calc_demo_program_with_parameters_reduction", "calc_demo_add_object_reduction", "const calc_demo_lexeme * left", "const calc_demo_lexeme * right", "calc_demo_typed_reducer", "calc_demo_typed_reducer_from_boxed", "calc_demo_parse_value_typed"} {
		if !strings.Contains(typedHeader, fragment) {
			t.Fatalf("parser_typed.h missing %q:\n%s", fragment, typedHeader)
		}
	}

	scannerSource := readGeneratedFile(t, out, "scanner.c")
	parserSource := readGeneratedFile(t, out, "parser.c")
	for _, fragment := range []string{`return "program.withParameters"`, `return "addObject"`} {
		if !strings.Contains(parserSource, fragment) {
			t.Fatalf("parser.c missing preserved action label %q:\n%s", fragment, parserSource)
		}
	}
	for _, source := range []string{scannerSource, parserSource} {
		if !strings.Contains(source, "lf_clear_error") {
			t.Fatalf("generated C source does not clear stale errors:\n%s", source)
		}
	}
	for _, fragment := range []string{"_stream_read_fn", "_stream_scanner", "_stream_scanner_next", "_stream_scanner_free"} {
		if !strings.Contains(readGeneratedFile(t, out, "scanner.h")+scannerSource, fragment) {
			t.Fatalf("generated C scanner missing stream fragment %q", fragment)
		}
	}
	if !strings.Contains(readGeneratedFile(t, out, "parser.h")+parserSource, "_stream_scanner_source_next") {
		t.Fatalf("generated C parser missing stream scanner source adapter")
	}
}

func TestCPrefixSanitizesNamesForPublicSymbols(t *testing.T) {
	tests := map[string]string{
		"":               "fallback",
		"calc-demo":      "calc_demo",
		"123 report":     "lf_123_report",
		"---":            "langforge_generated",
		"Vehicle_Report": "vehicle_report",
	}
	for input, want := range tests {
		if got := cPrefix(input, "fallback"); got != want {
			t.Fatalf("cPrefix(%q) = %q, want %q", input, got, want)
		}
	}
}

func readGeneratedFile(t *testing.T, dir string, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
