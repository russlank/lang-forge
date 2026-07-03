package cpp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
)

func TestGenerateWritesConventionalCppFilesAndMetadata(t *testing.T) {
	parsed, diagnostics := spec.ParseCombined([]byte(`%target cpp
%package langforge::examples::calc
%semantic cpp mode reducer
%semantic cpp type S double
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : left=A right=B {cpp: program.withParameters}
  | left=B right=A {cpp: addObject}
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
		"tokens.hpp",
		"scanner.hpp",
		"scanner.cpp",
		"parser.hpp",
		"parser_typed.hpp",
		"parser.cpp",
	} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	manifest := readGeneratedFile(t, out, "langforge.manifest.json")
	for _, fragment := range []string{`"target": "cpp"`, `"namespace": "langforge::examples::calc"`, `"cppConstant": "ProgramWithParameters"`, `"cppConstant": "AddObject"`} {
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

	tokens := readGeneratedFile(t, out, "tokens.hpp")
	for _, fragment := range []string{"enum class Token", "A = 2", "inline std::string_view token_name"} {
		if !strings.Contains(tokens, fragment) {
			t.Fatalf("tokens.hpp missing %q:\n%s", fragment, tokens)
		}
	}

	parserHeader := readGeneratedFile(t, out, "parser.hpp")
	for _, fragment := range []string{"enum class SemanticAction", "ProgramWithParameters", "AddObject", "std::vector<std::string_view> labels", "value_for", "class ReducerMap", "validate_coverage", "struct ParseResult", "class ParseError", "parse_recovering"} {
		if !strings.Contains(parserHeader, fragment) {
			t.Fatalf("parser.hpp missing %q:\n%s", fragment, parserHeader)
		}
	}

	typedHeader := readGeneratedFile(t, out, "parser_typed.hpp")
	for _, fragment := range []string{"struct ProgramWithParametersReduction", "struct AddObjectReduction", "Lexeme left", "Lexeme right", "using ProgramWithParametersHandler", "typed_program_with_parameters", "typed_add_object", "typed_reducer_map_from_boxed", "empty std::any"} {
		if !strings.Contains(typedHeader, fragment) {
			t.Fatalf("parser_typed.hpp missing %q:\n%s", fragment, typedHeader)
		}
	}

	parserSource := readGeneratedFile(t, out, "parser.cpp")
	for _, fragment := range []string{"std::lower_bound", "semantic_action_lookup", `"program.withParameters"`, `"addObject"`, "ParserActionKind", "label_symbols", "validate_coverage", "unknown reducer registered"} {
		if !strings.Contains(parserSource, fragment) {
			t.Fatalf("parser.cpp missing %q:\n%s", fragment, parserSource)
		}
	}
	if strings.Contains(parserSource, "switch (") {
		t.Fatalf("C++ parser source should prefer table/map dispatch over switch:\n%s", parserSource)
	}
}

func TestCppNamespaceValidationAndFallback(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		outBase string
		want    string
		wantErr bool
	}{
		{name: "explicit double colon", pkg: "langforge::examples::calc", outBase: "generated", want: "langforge::examples::calc"},
		{name: "explicit dotted", pkg: "LangForge.Examples.Calc", outBase: "generated", want: "LangForge::Examples::Calc"},
		{name: "fallback", pkg: "", outBase: "calc-generated", want: "LangForge::Generated::CalcGenerated"},
		{name: "keyword", pkg: "class", outBase: "generated", wantErr: true},
		{name: "bad dash", pkg: "bad-name", outBase: "generated", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cppNamespace(tt.pkg, tt.outBase)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, "::") != tt.want {
				t.Fatalf("namespace = %q, want %q", strings.Join(got, "::"), tt.want)
			}
		})
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
