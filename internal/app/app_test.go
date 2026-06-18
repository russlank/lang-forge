package app

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidate_CalcSpec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", filepath.Join("..", "..", "examples", "go", "calc", "calc.lf")}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunValidate_UCDTCalcSplitFiles(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"validate",
		"--lex", filepath.Join("..", "..", "testdata", "ucdt", "calc", "calc.l"),
		"--yacc", filepath.Join("..", "..", "testdata", "ucdt", "calc", "calc.y"),
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunValidate_UCDTDrawSplitFiles(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"validate",
		"--lex", filepath.Join("..", "..", "testdata", "ucdt", "draw", "draw.l"),
		"--yacc", filepath.Join("..", "..", "testdata", "ucdt", "draw", "draw.y"),
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunValidate_UCDTMetaSplitFiles(t *testing.T) {
	tests := []struct {
		name string
		lex  string
		yacc string
	}{
		{name: "lex-tool", lex: "lex.l", yacc: "lex.y"},
		{name: "yacc-tool", lex: "yacc.l", yacc: "yacc.y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), []string{
				"validate",
				"--lex", filepath.Join("..", "..", "testdata", "ucdt", "metas", tt.lex),
				"--yacc", filepath.Join("..", "..", "testdata", "ucdt", "metas", tt.yacc),
			}, &stdout, &stderr)
			if code != ExitOK {
				t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "ok:") {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

func TestRunValidate_ConflictExit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ambig.lf")
	writeFile(t, path, `%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : S S | A ;
`)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", path}, &stdout, &stderr)
	if code != ExitConflict {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "conflict") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunValidate_DefaultLALRAcceptsGrammarThatSLRRejects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lr1.lf")
	writeFile(t, path, lr1ButNotSLRSpec(""))
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", path}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("default LALR exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	slrPath := filepath.Join(dir, "slr.lf")
	writeFile(t, slrPath, lr1ButNotSLRSpec("%type slr\n"))
	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"validate", "--spec", slrPath}, &stdout, &stderr)
	if code != ExitConflict {
		t.Fatalf("SLR exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestRunValidate_IELRAcceptsGrammarWithLALRMergeConflict(t *testing.T) {
	dir := t.TempDir()
	lalrPath := filepath.Join(dir, "mysterious-lalr.lf")
	writeFile(t, lalrPath, mysteriousConflictSpec("%type lalr\n"))
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", lalrPath}, &stdout, &stderr)
	if code != ExitConflict {
		t.Fatalf("LALR exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "conflict") {
		t.Fatalf("LALR stderr = %q", stderr.String())
	}

	for _, algorithm := range []string{"ielr", "canonical"} {
		path := filepath.Join(dir, algorithm+".lf")
		writeFile(t, path, mysteriousConflictSpec("%type "+algorithm+"\n"))
		stdout.Reset()
		stderr.Reset()
		code = Run(context.Background(), []string{"validate", "--spec", path}, &stdout, &stderr)
		if code != ExitOK {
			t.Fatalf("%s exit = %d, stdout=%s stderr=%s", algorithm, code, stdout.String(), stderr.String())
		}
	}
}

func TestRunValidate_RejectsNullableLexerRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-rule.lf")
	writeFile(t, path, `%token A
%start S
%% lexer
"" => token(A);
%% parser
S : A ;
`)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", path}, &stdout, &stderr)
	if code != ExitValidate {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "LF206") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunValidate_RejectsTokenNonterminalCollision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "collision.lf")
	writeFile(t, path, `%token S A
%start S
%% lexer
"a" => token(A);
%% parser
S : A ;
`)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", path}, &stdout, &stderr)
	if code != ExitValidate {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "LF303") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunInspect_TextReportsSelectedAlgorithm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ielr.lf")
	writeFile(t, path, `%type ielr
%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A ;
`)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"inspect", "--spec", path, "--format", "text"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Parser algorithm: ielr") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGenerate_WritesDeterministicManifestAndTokens(t *testing.T) {
	out := t.TempDir()
	specPath := filepath.Join("..", "..", "examples", "go", "calc", "calc.lf")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	manifest1 := readFile(t, filepath.Join(out, "langforge.manifest.json"))
	if !strings.Contains(manifest1, `"encoding": "utf8"`) || !strings.Contains(manifest1, `"domain"`) || !strings.Contains(manifest1, `"SemanticActionAdd"`) {
		t.Fatalf("manifest does not record scanner metadata:\n%s", manifest1)
	}
	tokens := readFile(t, filepath.Join(out, "tokens.go"))
	if !strings.Contains(tokens, "type Token int") || !strings.Contains(tokens, "TokenNumber") {
		t.Fatalf("unexpected tokens.go:\n%s", tokens)
	}
	scanner := readFile(t, filepath.Join(out, "scanner.go"))
	if !strings.Contains(scanner, "// Source: "+specPath) {
		t.Fatalf("scanner.go does not record source file:\n%s", scanner)
	}
	parser := readFile(t, filepath.Join(out, "parser.go"))
	if !strings.Contains(parser, "// Source: "+specPath) || !strings.Contains(parser, "// Source: "+specPath+":") {
		t.Fatalf("parser.go does not record source references:\n%s", parser)
	}
	for _, name := range []string{"scanner.go", "parser.go", "langforge.tables.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected generated %s: %v", name, err)
		}
	}
	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("second exit = %d, stderr = %s", code, stderr.String())
	}
	manifest2 := readFile(t, filepath.Join(out, "langforge.manifest.json"))
	if manifest1 != manifest2 {
		t.Fatalf("manifest changed:\n%s\n---\n%s", manifest1, manifest2)
	}
}

func TestRunGenerate_WritesCScannerParserFiles(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "semantic-c.lf")
	writeFile(t, specPath, `%target c
%package semantic-c
%semantic c mode reducer
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : A B {c: pair} ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "c", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "generated ") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, name := range []string{"tokens.h", "scanner.h", "scanner.c", "parser.h", "parser.c", "langforge.manifest.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected C generated file %s: %v", name, err)
		}
	}
	manifest := readFile(t, filepath.Join(out, "langforge.manifest.json"))
	if !strings.Contains(manifest, `"target": "c"`) || !strings.Contains(manifest, `"prefix": "semantic_c"`) {
		t.Fatalf("unexpected C manifest:\n%s", manifest)
	}
	parserHeader := readFile(t, filepath.Join(out, "parser.h"))
	if !strings.Contains(parserHeader, "SEMANTIC_C_ACTION_PAIR") {
		t.Fatalf("parser.h does not expose C semantic action enum:\n%s", parserHeader)
	}
}

func TestRunGenerate_RejectsInvalidGoPackageName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-package.lf")
	writeFile(t, path, `%package bad-name
%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A ;
`)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", path, "--target", "go", "--out", filepath.Join(dir, "out")}, &stdout, &stderr)
	if code != ExitIO {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid Go package name") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunGenerate_GeneratedGoScannerParserCompilesAndParses(t *testing.T) {
	goBin := "/usr/local/go/bin/go"
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("go binary unavailable at %s", goBin)
	}
	out := t.TempDir()
	specPath := filepath.Join("..", "..", "examples", "go", "calc", "calc.lf")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	writeFile(t, filepath.Join(out, "generated_test.go"), `package calc

import (
	"sync"
	"testing"
)

func TestGeneratedScannerParser(t *testing.T) {
	tokens, err := Tokenize("1+2*(3-4)")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 9 {
		t.Fatalf("tokens len = %d", len(tokens))
	}
	if tokens[0].Token != TokenNumber || tokens[1].Token != TokenPlus {
		t.Fatalf("unexpected tokens: %#v", tokens[:2])
	}
	if err := Parse(tokens); err != nil {
		t.Fatal(err)
	}
	withEOF := append(append([]Lexeme{}, tokens...), Lexeme{Token: TokenEOF})
	if err := Parse(withEOF); err != nil {
		t.Fatalf("explicit EOF should be accepted: %v", err)
	}
	trailingAfterEOF := append(append([]Lexeme{}, withEOF...), Lexeme{Token: TokenPlus, Text: "+"})
	if err := Parse(trailingAfterEOF); err == nil {
		t.Fatal("expected token-after-EOF parse error")
	}
	bad, err := Tokenize("1+")
	if err != nil {
		t.Fatal(err)
	}
	if err := Parse(bad); err == nil {
		t.Fatal("expected parse error for incomplete expression")
	}
	if _, err := Tokenize("1@"); err == nil {
		t.Fatal("expected scanner error for unmatched input")
	}
}

func TestGeneratedScannerParserConcurrentUse(t *testing.T) {
	parser := NewParser()
	input := "1+2*(3-4)"
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tokens, err := Tokenize(input)
			if err != nil {
				errs <- err
				return
			}
			if err := parser.Parse(tokens); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	expected, err := Tokenize(input)
	if err != nil {
		t.Fatal(err)
	}
	shared := NewScanner(input)
	var mu sync.Mutex
	count := 0
	errs = make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, ok, err := shared.Next()
				if err != nil {
					errs <- err
					return
				}
				if !ok {
					return
				}
				mu.Lock()
				count++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if count != len(expected) {
		t.Fatalf("shared scanner token count = %d, want %d", count, len(expected))
	}
}
`)
	run(t, out, goBin, "mod", "init", "calc")
	run(t, out, goBin, "test", "./...")
}

func TestRunGenerate_GeneratedGoScannerTokenizesUTF8(t *testing.T) {
	goBin := "/usr/local/go/bin/go"
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("go binary unavailable at %s", goBin)
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "unicode.lf")
	writeFile(t, specPath, `%package unicodelex
%scanner utf8
%token Word
%start S
%% lexer
\p{L}+ => token(Word);
[1-32]+ => skip;
%% parser
S : Word ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	writeFile(t, filepath.Join(out, "generated_test.go"), `package unicodelex

import "testing"

func TestGeneratedUTF8Scanner(t *testing.T) {
	tokens, err := Tokenize("åβ")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 {
		t.Fatalf("tokens = %#v", tokens)
	}
	if tokens[0].Text != "åβ" || tokens[0].Start != 0 || tokens[0].End != len("åβ") {
		t.Fatalf("token span = %#v", tokens[0])
	}
	if tokens[0].StartLine != 1 || tokens[0].StartColumn != 1 || tokens[0].EndColumn != 3 {
		t.Fatalf("token position = %#v", tokens[0])
	}
	if err := Parse(tokens); err != nil {
		t.Fatal(err)
	}
	if _, err := Tokenize(string([]byte{0xff})); err == nil {
		t.Fatal("expected invalid UTF-8 scanner error")
	}
}
`)
	run(t, out, goBin, "mod", "init", "unicodelex")
	run(t, out, goBin, "test", "./...")
}

func TestRunGenerate_GeneratedGoParserDispatchesSemanticReducer(t *testing.T) {
	goBin := "/usr/local/go/bin/go"
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("go binary unavailable at %s", goBin)
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "semantic.lf")
	writeFile(t, specPath, `%package semantic
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : A B {go: pair} ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	writeFile(t, filepath.Join(out, "generated_test.go"), `package semantic

import (
	"errors"
	"strings"
	"testing"
)

func TestGeneratedReducerDispatch(t *testing.T) {
	tokens, err := Tokenize("ab")
	if err != nil {
		t.Fatal(err)
	}
	if err := Parse(tokens); err != nil {
		t.Fatalf("recognizer parse should still work without a reducer: %v", err)
	}
	defaultValue, err := ParseValue(tokens)
	if err != nil {
		t.Fatal(err)
	}
	defaultItems, ok := defaultValue.([]Value)
	if !ok || len(defaultItems) != 2 {
		t.Fatalf("default value = %#v, want two shifted values", defaultValue)
	}
	value, err := ParseWithReducer(tokens, ReducerFunc(func(ctx Reduction) (Value, error) {
		if ctx.Rule != 1 || ctx.LHS != "S" || ctx.Action != "pair" || ctx.ActionID != SemanticActionPair {
			t.Fatalf("unexpected reduction context: %#v", ctx)
		}
		if len(ctx.RHS) != 2 || ctx.RHS[0] != "A" || ctx.RHS[1] != "B" {
			t.Fatalf("rhs = %#v", ctx.RHS)
		}
		if len(ctx.Values) != 2 {
			t.Fatalf("values = %#v", ctx.Values)
		}
		left := ctx.Values[0].(Lexeme)
		right := ctx.Values[1].(Lexeme)
		return left.Text + right.Text, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if value != "ab" {
		t.Fatalf("value = %#v, want ab", value)
	}
	action, ok := LookupSemanticAction("pair")
	if !ok || action != SemanticActionPair || action.String() != "pair" {
		t.Fatalf("semantic action lookup = %v, %v", action, ok)
	}
	value, err = ParseWithReducer(tokens, ReducerMap{
		SemanticActionPair: func(ctx Reduction) (Value, error) {
			return ctx.ActionID.String(), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value != "pair" {
		t.Fatalf("map reducer value = %#v, want pair", value)
	}
	_, err = ParseWithReducer(tokens, ReducerFunc(func(ctx Reduction) (Value, error) {
		return nil, errors.New("reducer stopped")
	}))
	if err == nil || !strings.Contains(err.Error(), "reducer stopped") {
		t.Fatalf("reducer error = %v", err)
	}
}
`)
	run(t, out, goBin, "mod", "init", "semantic")
	run(t, out, goBin, "test", "./...")
}

func TestRunGenerate_GeneratedGoParserSupportsInlineSemanticImports(t *testing.T) {
	goBin := "/usr/local/go/bin/go"
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("go binary unavailable at %s", goBin)
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "semantic-inline.lf")
	writeFile(t, specPath, `%package generated
%semantic go mode inline
%semantic go import sem "semanticinline/semantics"
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : A B {go:
	return sem.JoinLexemeText(ctx.Values[0], ctx.Values[1])
} ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if manifest := readFile(t, filepath.Join(out, "langforge.manifest.json")); !strings.Contains(manifest, `"mode": "inline"`) && !strings.Contains(manifest, `"inline"`) {
		t.Fatalf("manifest does not record inline semantics:\n%s", manifest)
	}
	parser := readFile(t, filepath.Join(out, "parser.go"))
	if !strings.Contains(parser, "//line "+specPath+":") {
		t.Fatalf("parser.go does not contain inline action line directive for %s:\n%s", specPath, parser)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module semanticinline\n\ngo 1.25.0\n")
	if err := os.Mkdir(filepath.Join(dir, "semantics"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "semantics", "semantics.go"), `package semantics

import "reflect"

func JoinLexemeText(left, right any) (any, error) {
	return reflect.ValueOf(left).FieldByName("Text").String() + reflect.ValueOf(right).FieldByName("Text").String(), nil
}
`)
	writeFile(t, filepath.Join(out, "generated_test.go"), `package generated

import "testing"

func TestGeneratedInlineSemanticImport(t *testing.T) {
	tokens, err := Tokenize("ab")
	if err != nil {
		t.Fatal(err)
	}
	value, err := ParseValue(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if value != "ab" {
		t.Fatalf("value = %#v, want ab", value)
	}
	if SemanticActionMode != "inline" {
		t.Fatalf("mode = %q", SemanticActionMode)
	}
	if len(SemanticIncludes) != 1 || SemanticIncludes[0].Alias != "sem" || SemanticIncludes[0].Path != "semanticinline/semantics" {
		t.Fatalf("includes = %#v", SemanticIncludes)
	}
}
`)
	run(t, dir, goBin, "test", "./...")
}

func TestRunGenerate_GeneratedCSharpScannerParserCompilesAndParses(t *testing.T) {
	dotnet, err := exec.LookPath("dotnet")
	if err != nil {
		t.Skip("dotnet unavailable")
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "calc.lf")
	writeFile(t, specPath, csharpCalcSpec())
	out := filepath.Join(dir, "Generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "csharp", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, name := range []string{"Tokens.cs", "Scanner.cs", "Parser.cs"} {
		writeFile(t, filepath.Join(out, name), "// stale legacy generated filename\n")
	}
	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "csharp", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("second generate exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	manifest := readFile(t, filepath.Join(out, "langforge.manifest.json"))
	if !strings.Contains(manifest, `"target": "csharp"`) || !strings.Contains(manifest, `"namespace": "LangForge.Examples.Calc.Generated"`) {
		t.Fatalf("unexpected C# manifest:\n%s", manifest)
	}
	for _, name := range []string{"Tokens.g.cs", "Scanner.g.cs", "Parser.g.cs"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("generated C# file %s missing: %v", name, err)
		}
	}
	for _, name := range []string{"Tokens.cs", "Scanner.cs", "Parser.cs"} {
		if _, err := os.Stat(filepath.Join(out, name)); !os.IsNotExist(err) {
			t.Fatalf("generated C# file %s should use .g.cs suffix; stat error = %v", name, err)
		}
	}
	writeFile(t, filepath.Join(dir, "CalcCSharp.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
</Project>
`)
	writeFile(t, filepath.Join(dir, "Program.cs"), `using System;
using System.Linq;
using System.Threading.Tasks;
using LangForge.Examples.Calc.Generated;

static double Eval(string source)
{
    var tokens = Scanner.Tokenize(source);
    var reducers = new ReducerMap
    {
        [SemanticAction.Start] = ctx => ctx.Values[0],
        [SemanticAction.Pass] = ctx => ctx.Values[0],
        [SemanticAction.Group] = ctx => ctx.Values[1],
        [SemanticAction.Number] = ctx => double.Parse(((Lexeme)ctx.Values[0]!).Text),
        [SemanticAction.Negate] = ctx => -(double)ctx.Values[1]!,
        [SemanticAction.Add] = ctx => (double)ctx.Values[0]! + (double)ctx.Values[2]!,
        [SemanticAction.Subtract] = ctx => (double)ctx.Values[0]! - (double)ctx.Values[2]!,
        [SemanticAction.Multiply] = ctx => (double)ctx.Values[0]! * (double)ctx.Values[2]!,
        [SemanticAction.Divide] = ctx => (double)ctx.Values[0]! / (double)ctx.Values[2]!,
    };
    return (double)Parser.ParseWithReducer(tokens, reducers)!;
}

static void Check(bool condition, string message)
{
    if (!condition)
    {
        throw new InvalidOperationException(message);
    }
}

Check(Math.Abs(Eval("1+2*(3-4)") - -1) < 0.0001, "wrong expression result");
var visible = Scanner.Tokenize("1+2");
Parser.Parse(visible);
Parser.Parse(visible.Concat(new[] { new Lexeme(Token.EOF, "", "", 0, 0, 1, 1, 1, 1) }).ToArray());
try
{
    Parser.Parse(visible.Concat(new[] {
        new Lexeme(Token.EOF, "", "", 0, 0, 1, 1, 1, 1),
        new Lexeme(Token.Plus, "+", "", 0, 1, 1, 1, 1, 2),
    }).ToArray());
    throw new InvalidOperationException("expected token-after-EOF parse error");
}
catch (InvalidOperationException ex) when (ex.Message.Contains("token after EOF"))
{
}
try
{
    Scanner.Tokenize("1@");
    throw new InvalidOperationException("expected scanner error");
}
catch (InvalidOperationException ex) when (ex.Message.Contains("no lexical rule"))
{
}
try
{
    Scanner.Tokenize("\ud800");
    throw new InvalidOperationException("expected invalid UTF-16 error");
}
catch (InvalidOperationException ex) when (ex.Message.Contains("invalid UTF-16"))
{
}

var parser = new Parser();
Parallel.For(0, 16, _ => parser.ParseInput(Scanner.Tokenize("1+2*(3-4)")));
var shared = new Scanner("1+2*(3-4)");
int count = 0;
Parallel.For(0, 4, _ =>
{
    while (shared.Next(out var _))
    {
        System.Threading.Interlocked.Increment(ref count);
    }
});
Check(count == Scanner.Tokenize("1+2*(3-4)").Count, $"shared scanner count {count}");
Console.WriteLine("ok");
`)
	run(t, dir, dotnet, "run", "--project", filepath.Join(dir, "CalcCSharp.csproj"))
}

func TestRunGenerate_InlineSemanticErrorsReportGrammarSource(t *testing.T) {
	goBin := "/usr/local/go/bin/go"
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("go binary unavailable at %s", goBin)
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "bad-inline.lf")
	writeFile(t, specPath, `%package generated
%semantic go mode inline
%token A
%start S
%% lexer
"a" => token(A);
%% parser
S : A {go:
	return MissingInlineHelper(ctx.Values[0])
} ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module badinline\n\ngo 1.25.0\n")
	cmd := exec.Command(goBin, "test", "./...")
	cmd.Dir = dir
	outBytes, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go test unexpectedly passed:\n%s", outBytes)
	}
	if !strings.Contains(string(outBytes), "bad-inline.lf") {
		t.Fatalf("compiler output did not refer to grammar source:\n%s", outBytes)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func lr1ButNotSLRSpec(prefix string) string {
	return prefix + `%token ID Star Eq
%start S
%% lexer
"id" => token(ID);
"*" => token(Star);
"=" => token(Eq);
[1-32]+ => skip;
%% parser
S : L Eq R | R ;
L : Star R | ID ;
R : L ;
`
}

func mysteriousConflictSpec(prefix string) string {
	return prefix + `%token ID Colon Comma
%start Def
%% lexer
"id" => token(ID);
":" => token(Colon);
"," => token(Comma);
[1-32]+ => skip;
%% parser
Def : ParamSpec ReturnSpec Comma ;
ParamSpec : Type | NameList Colon Type ;
ReturnSpec : Type | Name Colon Type ;
Type : ID ;
Name : ID ;
NameList : Name | Name Comma NameList ;
`
}

func csharpCalcSpec() string {
	return `%target csharp
%package LangForge.Examples.Calc.Generated
%token Number Plus Minus Mul Div LParen RParen
%start S
%% lexer
DIGIT = [0-9];
NUMBER = DIGIT+;
NUMBER => token(Number);
"+" => token(Plus);
"-" => token(Minus);
"*" => token(Mul);
"/" => token(Div);
"(" => token(LParen);
")" => token(RParen);
[1-32]+ => skip;
%% parser
S : Expr {csharp: start} ;
Expr : Expr Plus Term {csharp: add}
     | Expr Minus Term {csharp: subtract}
     | Term {csharp: pass}
     ;
Term : Term Mul Factor {csharp: multiply}
     | Term Div Factor {csharp: divide}
     | Factor {csharp: pass}
     ;
Factor : Number {csharp: number}
       | LParen Expr RParen {csharp: group}
       | Minus Factor {csharp: negate}
       ;
`
}
