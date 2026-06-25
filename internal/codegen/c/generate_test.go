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
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : A B {c: pair.value} ;
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
		"parser.c",
	} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	manifest := readGeneratedFile(t, out, "langforge.manifest.json")
	for _, fragment := range []string{`"target": "c"`, `"prefix": "calc_demo"`, `"CALC_DEMO_ACTION_PAIR_VALUE"`} {
		if !strings.Contains(manifest, fragment) {
			t.Fatalf("manifest missing %q:\n%s", fragment, manifest)
		}
	}
	actionManifest := readGeneratedFile(t, out, "langforge.actions.json")
	for _, fragment := range []string{`"name": "pair.value"`, `"lhs": "S"`, `"symbol": "A"`} {
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
	for _, fragment := range []string{"typedef enum calc_demo_semantic_action", "CALC_DEMO_ACTION_PAIR_VALUE"} {
		if !strings.Contains(parserHeader, fragment) {
			t.Fatalf("parser.h missing %q:\n%s", fragment, parserHeader)
		}
	}

	scannerSource := readGeneratedFile(t, out, "scanner.c")
	parserSource := readGeneratedFile(t, out, "parser.c")
	for _, source := range []string{scannerSource, parserSource} {
		if !strings.Contains(source, "lf_clear_error") {
			t.Fatalf("generated C source does not clear stale errors:\n%s", source)
		}
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
