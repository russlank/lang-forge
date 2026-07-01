package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestRunValidate_VerbosityReportsBuildDecisions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	specPath := filepath.Join("..", "..", "examples", "go", "calc", "calc.lf")
	code := Run(context.Background(), []string{"validate", "--spec", specPath, "--verbosity", "2"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, fragment := range []string{
		"[lf] load: combined spec=",
		"[lf] lexer: DFA states=",
		"[lf] grammar rule",
		"[lf] parser: table algorithm=",
		"[lf] parser: actionKinds",
	} {
		if !strings.Contains(stderr.String(), fragment) {
			t.Fatalf("stderr missing %q:\n%s", fragment, stderr.String())
		}
	}
}

func TestRunValidate_RejectsInvalidVerbosity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"validate", "--spec", filepath.Join("..", "..", "examples", "go", "calc", "calc.lf"), "--verbosity", "4"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--verbosity must be between 0 and 3") {
		t.Fatalf("stderr = %q", stderr.String())
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
	if !strings.Contains(stderr.String(), "hint: shift/reduce conflict") ||
		!strings.Contains(stderr.String(), "state items:") ||
		!strings.Contains(stderr.String(), "S -> S S •") ||
		!strings.Contains(stderr.String(), "ambig.lf:") {
		t.Fatalf("stderr does not include source-rich conflict context:\n%s", stderr.String())
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
	if !strings.Contains(stdout.String(), "IELR state counts:") || !strings.Contains(stdout.String(), "IELR merges:") {
		t.Fatalf("stdout does not include IELR report:\n%s", stdout.String())
	}
}

func TestRunInspect_JSONKeepsStdoutCleanWithVerbosity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"inspect", "--spec", filepath.Join("..", "..", "examples", "go", "calc", "calc.lf"), "--format", "json", "--verbose"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var summary Summary
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("inspect JSON did not decode: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if summary.ParseTable == nil || len(summary.ParseTable.States) == 0 {
		t.Fatalf("inspect JSON missing parse table:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "[lf] load:") || strings.Contains(stdout.String(), "[lf]") {
		t.Fatalf("verbosity should write only to stderr, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunInspect_IELRReportsMergeDecisionsInTextAndJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mysterious-ielr.lf")
	writeFile(t, path, mysteriousConflictSpec("%type ielr\n"))

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"inspect", "--spec", path, "--format", "text"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("text inspect exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	text := stdout.String()
	for _, fragment := range []string{
		"IELR state counts: LALR=",
		"IELR merges: accepted=",
		"rejected core",
		"action-conflict",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("text inspect missing %q:\n%s", fragment, text)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"inspect", "--spec", path, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("json inspect exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var summary Summary
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("inspect JSON did not decode: %v\n%s", err, stdout.String())
	}
	if summary.ParseTable.IELR == nil {
		t.Fatalf("inspect JSON missing IELR report:\n%s", stdout.String())
	}
	if summary.ParseTable.IELR.LALRStates >= summary.ParseTable.IELR.IELRStates || summary.ParseTable.IELR.IELRStates > summary.ParseTable.IELR.CanonicalStates {
		t.Fatalf("unexpected IELR state counts: %#v", summary.ParseTable.IELR)
	}
	if len(summary.ParseTable.IELR.RejectedMerges) == 0 {
		t.Fatalf("expected rejected merge details: %#v", summary.ParseTable.IELR)
	}
}

func TestRunInspect_IELRJSONIsStableAcrossBuilds(t *testing.T) {
	specPath := filepath.Join("..", "..", "examples", "parser-algorithms", "mysterious-conflict-ielr.lf")
	var firstOut, firstErr bytes.Buffer
	code := Run(context.Background(), []string{"inspect", "--spec", specPath, "--format", "json"}, &firstOut, &firstErr)
	if code != ExitOK {
		t.Fatalf("first inspect exit = %d, stdout=%s stderr=%s", code, firstOut.String(), firstErr.String())
	}

	var secondOut, secondErr bytes.Buffer
	code = Run(context.Background(), []string{"inspect", "--spec", specPath, "--format", "json"}, &secondOut, &secondErr)
	if code != ExitOK {
		t.Fatalf("second inspect exit = %d, stdout=%s stderr=%s", code, secondOut.String(), secondErr.String())
	}
	if firstOut.String() != secondOut.String() {
		t.Fatalf("IELR inspect JSON changed between builds")
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
	if !strings.Contains(parser, "Grammar rule") || !strings.Contains(parser, "{go: add}") {
		t.Fatalf("parser.go does not annotate generated tables with grammar rules:\n%s", parser)
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

func TestRunGenerate_VerbosityReportsGenerationStages(t *testing.T) {
	out := t.TempDir()
	specPath := filepath.Join("..", "..", "examples", "go", "calc", "calc.lf")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out, "-v", "1"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, fragment := range []string{
		"[lf] load:",
		"[lf] parser: table algorithm=",
		"[lf] generate: target=go",
		"[lf] generate: completed target=go",
	} {
		if !strings.Contains(stderr.String(), fragment) {
			t.Fatalf("stderr missing %q:\n%s", fragment, stderr.String())
		}
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
	for _, name := range []string{"tokens.h", "scanner.h", "scanner.c", "parser.h", "parser.c", "langforge.manifest.json", "langforge.actions.json"} {
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
	parserSource := readFile(t, filepath.Join(out, "parser.c"))
	if !strings.Contains(parserSource, "Grammar rule") || !strings.Contains(parserSource, "S -> A B {c: pair}") || !strings.Contains(parserSource, "Source: "+specPath+":") {
		t.Fatalf("parser.c does not annotate generated tables with source grammar rules:\n%s", parserSource)
	}
}

func TestRunGenerate_WritesCppScannerParserFiles(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "semantic-cpp.lf")
	writeFile(t, specPath, `%target cpp
%package langforge::tests::semantic
%semantic cpp mode reducer
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : A B {cpp: pair} ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "c++", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "generated ") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	for _, name := range []string{"tokens.hpp", "scanner.hpp", "scanner.cpp", "parser.hpp", "parser.cpp", "langforge.manifest.json", "langforge.actions.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected C++ generated file %s: %v", name, err)
		}
	}
	manifest := readFile(t, filepath.Join(out, "langforge.manifest.json"))
	if !strings.Contains(manifest, `"target": "cpp"`) || !strings.Contains(manifest, `"namespace": "langforge::tests::semantic"`) {
		t.Fatalf("unexpected C++ manifest:\n%s", manifest)
	}
	parserHeader := readFile(t, filepath.Join(out, "parser.hpp"))
	for _, fragment := range []string{"enum class SemanticAction", "Pair"} {
		if !strings.Contains(parserHeader, fragment) {
			t.Fatalf("parser.hpp does not expose C++ semantic action fragment %q:\n%s", fragment, parserHeader)
		}
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

func TestRunGenerate_GeneratedGoParserRecoversAndReportsExpectedTokens(t *testing.T) {
	goBin := "/usr/local/go/bin/go"
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("go binary unavailable at %s", goBin)
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "recovery.lf")
	writeFile(t, specPath, `%target go
%package recovery
%token Ident Number Assign Semi
%alias Ident "identifier"
%alias Number "number literal"
%group value Ident Number
%hide-expected Semi
%% lexer
IDENT = [A-Za-z_] [A-Za-z0-9_]*;
NUMBER = [0-9]+;
IDENT => token(Ident);
NUMBER => token(Number);
"=" => token(Assign);
";" => token(Semi);
[1-32]+ => skip;
%% parser
Program : Statements ;
Statements : Statement Statements | %empty ;
Statement : Ident Assign Number Semi | error Semi ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	writeFile(t, filepath.Join(out, "recovery_test.go"), `package recovery

import (
	"errors"
	"testing"
)

func TestRecoveryAndExpectedTokens(t *testing.T) {
	tokens, err := Tokenize("x=y; y=2; z=; w=3;")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ParseRecovering(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted || len(result.Diagnostics) != 2 {
		t.Fatalf("result = %#v", result)
	}
	first := result.Diagnostics[0]
	if first.Unexpected != "Ident" || first.UnexpectedDisplay != "identifier" || first.StartLine != 1 || first.StartColumn != 3 {
		t.Fatalf("first diagnostic = %#v", first)
	}
	if len(first.Expected) != 1 || first.Expected[0].Display != "number literal" {
		t.Fatalf("expected tokens = %#v", first.Expected)
	}
	if first.Recovery.Kind != "recovered" || first.Recovery.Discarded != 1 {
		t.Fatalf("recovery = %#v", first.Recovery)
	}
	_, err = ParseValue(tokens)
	var parseErr *ParseError
	if !errors.As(err, &parseErr) || len(parseErr.Diagnostics) != 2 {
		t.Fatalf("ParseValue error = %#v", err)
	}
}

func TestUnrecoverableInputTerminates(t *testing.T) {
	tokens, err := Tokenize("=")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ParseRecovering(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted || len(result.Diagnostics) != 1 || result.Diagnostics[0].Recovery.Kind != "abort" {
		t.Fatalf("result = %#v", result)
	}
}
`)
	run(t, out, goBin, "mod", "init", "recovery")
	run(t, out, goBin, "test", "./...")
}

func TestRunGenerate_GeneratedCParserRecoversAndReportsExpectedTokens(t *testing.T) {
	cc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "recovery.lf")
	writeFile(t, specPath, recoverySpec("c", "recovery"))
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "c", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	writeFile(t, filepath.Join(dir, "main.c"), `#include "generated/parser.h"

#include <stdio.h>
#include <string.h>

static int require(int condition, const char *message) {
    if (!condition) {
        fprintf(stderr, "%s\n", message);
        return 0;
    }
    return 1;
}

int main(void) {
    recovery_error error = {{0}};
    recovery_lexeme *tokens = NULL;
    size_t count = 0;
    recovery_parse_result result;
    recovery_parse_result_init(&result);
    if (!recovery_tokenize("x=y; y=2; z=; w=3;", &tokens, &count, &error)) { return 1; }
    if (!recovery_parse_recovering(tokens, count, &result, &error)) { return 2; }
    if (!require(result.accepted && result.diagnostic_count == 2, "wrong recovery result")) { return 3; }
    const recovery_parse_diagnostic *first = &result.diagnostics[0];
    if (!require(strcmp(first->unexpected, "Ident") == 0 && strcmp(first->unexpected_display, "identifier") == 0 && first->start_column == 3, "wrong first diagnostic")) { return 4; }
    if (!require(first->expected_count == 1 && strcmp(first->expected[0].display, "number literal") == 0, "wrong expected token")) { return 5; }
    if (!require(strcmp(first->recovery, "recovered") == 0 && first->discarded == 1, "wrong recovery action")) { return 6; }
    recovery_parse_result_free(&result);
    if (recovery_parse(tokens, count, &error) || strstr(error.message, "parse error at 1:3") == NULL) { return 7; }
    recovery_free_lexemes(tokens);

    tokens = NULL;
    count = 0;
    if (!recovery_tokenize("=", &tokens, &count, &error)) { return 8; }
    if (!recovery_parse_recovering(tokens, count, &result, &error)) { return 9; }
    if (!require(!result.accepted && result.diagnostic_count == 1 && strcmp(result.diagnostics[0].recovery, "abort") == 0, "unrecoverable input did not terminate")) { return 10; }
    recovery_parse_result_free(&result);
    recovery_free_lexemes(tokens);
    return 0;
}
`)
	run(t, dir, cc, "-std=c11", "-Wall", "-Wextra", "-Werror", "-pedantic", "-Igenerated", "generated/scanner.c", "generated/parser.c", "main.c", "-o", "recovery-c")
	run(t, dir, filepath.Join(dir, "recovery-c"))
}

func TestRunGenerate_GeneratedCppParserRecoversAndReportsExpectedTokens(t *testing.T) {
	compilers := findCppCompilers()
	if len(compilers) == 0 {
		t.Skip("C++ compiler unavailable")
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "recovery.lf")
	writeFile(t, specPath, recoverySpec("cpp", "Recovery::Generated"))
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "cpp", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	writeFile(t, filepath.Join(dir, "main.cpp"), `#include "generated/parser.hpp"

#include <stdexcept>
#include <string>

using namespace Recovery::Generated;

static void require(bool condition, const char* message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

int main() {
    const auto tokens = tokenize("x=y; y=2; z=; w=3;");
    const auto result = parse_recovering(tokens);
    require(result.accepted && result.diagnostics.size() == 2, "wrong recovery result");
    const auto& first = result.diagnostics.front();
    require(first.unexpected == "Ident" && first.unexpected_display == "identifier" && first.start_column == 3, "wrong first diagnostic");
    require(first.expected.size() == 1 && first.expected.front().display == "number literal", "wrong expected token");
    require(first.recovery.kind == "recovered" && first.recovery.discarded == 1, "wrong recovery action");
    try {
        parse(tokens);
        throw std::runtime_error("compatibility parse should fail");
    } catch (const ParseError& error) {
        require(error.diagnostics().size() == 2, "wrong ParseError diagnostics");
    }
    const auto aborted = parse_recovering(tokenize("="));
    require(!aborted.accepted && aborted.diagnostics.size() == 1 && aborted.diagnostics.front().recovery.kind == "abort", "unrecoverable input did not terminate");
    return 0;
}
`)
	for index, cxx := range compilers {
		binary := fmt.Sprintf("recovery-cpp-%d", index)
		run(t, dir, cxx, "-std=c++17", "-Wall", "-Wextra", "-Werror", "-pedantic", "-Igenerated", "generated/scanner.cpp", "generated/parser.cpp", "main.cpp", "-o", binary)
		run(t, dir, filepath.Join(dir, binary))
	}
}

func TestRunGenerate_GeneratedCSharpParserRecoversAndReportsExpectedTokens(t *testing.T) {
	dotnet, err := exec.LookPath("dotnet")
	if err != nil {
		t.Skip("dotnet unavailable")
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "recovery.lf")
	writeFile(t, specPath, recoverySpec("csharp", "Recovery.Generated"))
	out := filepath.Join(dir, "Generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "csharp", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	writeFile(t, filepath.Join(dir, "Recovery.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
</Project>
`)
	writeFile(t, filepath.Join(dir, "Program.cs"), `using Recovery.Generated;

static void Require(bool condition, string message)
{
    if (!condition) throw new InvalidOperationException(message);
}

var tokens = Scanner.Tokenize("x=y; y=2; z=; w=3;");
var result = Parser.ParseRecovering(tokens);
Require(result.Accepted && result.Diagnostics.Count == 2, "wrong recovery result");
var first = result.Diagnostics[0];
Require(first.Unexpected == "Ident" && first.UnexpectedDisplay == "identifier" && first.StartColumn == 3, "wrong first diagnostic");
Require(first.Expected.Count == 1 && first.Expected[0].Display == "number literal", "wrong expected token");
Require(first.Recovery.Kind == "recovered" && first.Recovery.Discarded == 1, "wrong recovery action");
try
{
    Parser.Parse(tokens);
    throw new InvalidOperationException("compatibility parse should fail");
}
catch (ParseException error)
{
    Require(error.Diagnostics.Count == 2, "wrong ParseException diagnostics");
}
var aborted = Parser.ParseRecovering(Scanner.Tokenize("="));
Require(!aborted.Accepted && aborted.Diagnostics.Count == 1 && aborted.Diagnostics[0].Recovery.Kind == "abort", "unrecoverable input did not terminate");
`)
	run(t, dir, dotnet, "run", "--project", "Recovery.csproj")
}

func TestRunGenerate_GeneratedCppScannerParserCompilesAndParses(t *testing.T) {
	cxx, ok := findCppCompiler()
	if !ok {
		t.Skip("C++ compiler unavailable")
	}
	dir := t.TempDir()
	specPath := filepath.Join(dir, "calc.lf")
	writeFile(t, specPath, cppCalcSpec())
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "cpp", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	parserSource := readFile(t, filepath.Join(out, "parser.cpp"))
	if !strings.Contains(parserSource, "Grammar rule") || !strings.Contains(parserSource, "{cpp: add}") || !strings.Contains(parserSource, "Source: "+specPath+":") {
		t.Fatalf("generated C++ parser does not annotate generated tables with source grammar rules:\n%s", parserSource)
	}
	writeFile(t, filepath.Join(dir, "main.cpp"), `#include "generated/parser.hpp"

#include <any>
#include <atomic>
#include <cmath>
#include <iostream>
#include <stdexcept>
#include <string>
#include <thread>
#include <vector>

using namespace LangForge::Examples::Calc::Generated;

static double number_arg(const Reduction& ctx, std::size_t index) {
    return std::any_cast<double>(ctx.values.at(index));
}

static Lexeme lexeme_arg(const Reduction& ctx, std::size_t index) {
    return std::any_cast<Lexeme>(ctx.values.at(index));
}

static double eval(const std::string& source) {
    auto tokens = tokenize(source);
    ReducerMap reducers{
        {SemanticAction::Start, [](const Reduction& ctx) -> Value { return ctx.values.at(0); }},
        {SemanticAction::Pass, [](const Reduction& ctx) -> Value { return ctx.values.at(0); }},
        {SemanticAction::Group, [](const Reduction& ctx) -> Value { return ctx.values.at(1); }},
        {SemanticAction::Number, [](const Reduction& ctx) -> Value {
            const auto lexeme = lexeme_arg(ctx, 0);
            return std::stod(std::string(lexeme.text));
        }},
        {SemanticAction::Negate, [](const Reduction& ctx) -> Value { return -number_arg(ctx, 1); }},
        {SemanticAction::Add, [](const Reduction& ctx) -> Value { return number_arg(ctx, 0) + number_arg(ctx, 2); }},
        {SemanticAction::Subtract, [](const Reduction& ctx) -> Value { return number_arg(ctx, 0) - number_arg(ctx, 2); }},
        {SemanticAction::Multiply, [](const Reduction& ctx) -> Value { return number_arg(ctx, 0) * number_arg(ctx, 2); }},
        {SemanticAction::Divide, [](const Reduction& ctx) -> Value { return number_arg(ctx, 0) / number_arg(ctx, 2); }},
    };
    return std::any_cast<double>(parse_value(tokens, reducers));
}

static void require(bool condition, const char* message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

int main() {
    require(std::fabs(eval("1+2*(3-4)") - -1.0) < 0.000001, "wrong expression result");
    auto visible = tokenize("1+2");
    parse(visible);
    auto with_eof = visible;
    with_eof.push_back(Lexeme{Token::End, "", "", 0, 0, 1, 1, 1, 1});
    parse(with_eof);
    with_eof.push_back(Lexeme{Token::Plus, "+", "", 0, 1, 1, 1, 1, 2});
    try {
        parse(with_eof);
        throw std::runtime_error("expected token-after-EOF parse error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("token after EOF") != std::string::npos, "wrong EOF error");
    }
    try {
        tokenize("1@");
        throw std::runtime_error("expected scanner error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("no lexical rule") != std::string::npos, "wrong scanner error");
    }
    try {
        tokenize(std::string("1") + static_cast<char>(0xff));
        throw std::runtime_error("expected invalid UTF-8 error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("invalid UTF-8") != std::string::npos, "wrong UTF-8 error");
    }

    Parser parser;
    std::vector<std::thread> workers;
    for (int i = 0; i < 8; ++i) {
        workers.emplace_back([&parser]() {
            parser.parse(tokenize("1+2*(3-4)"));
        });
    }
    for (auto& worker : workers) {
        worker.join();
    }

    Scanner shared("1+2*(3-4)");
    std::atomic<int> count{0};
    workers.clear();
    for (int i = 0; i < 4; ++i) {
        workers.emplace_back([&shared, &count]() {
            Lexeme lexeme;
            while (shared.next(lexeme)) {
                ++count;
            }
        });
    }
    for (auto& worker : workers) {
        worker.join();
    }
    require(count == static_cast<int>(tokenize("1+2*(3-4)").size()), "shared scanner token count mismatch");
    SemanticAction action = SemanticAction::None;
    require(lookup_semantic_action("add", action) && action == SemanticAction::Add, "semantic action lookup failed");
    std::cout << "ok\n";
}
`)
	run(t, dir, cxx, "-std=c++17", "-Wall", "-Wextra", "-Werror", "-I", out, filepath.Join(dir, "main.cpp"), filepath.Join(out, "scanner.cpp"), filepath.Join(out, "parser.cpp"), "-o", filepath.Join(dir, "calc-cpp-test"))
	run(t, dir, filepath.Join(dir, "calc-cpp-test"))
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
%semantic go type S string
%token A B
%start S
%% lexer
"a" => token(A);
"b" => token(B);
[1-32]+ => skip;
%% parser
S : left=A right=B {go: pair} ;
`)
	out := filepath.Join(dir, "generated")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"generate", "--spec", specPath, "--target", "go", "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	actionManifest := readFile(t, filepath.Join(out, "langforge.actions.json"))
	for _, fragment := range []string{
		`"name": "pair"`,
		`"returnType": "string"`,
		`"label": "left"`,
		`"label": "right"`,
		`"typed": true`,
	} {
		if !strings.Contains(actionManifest, fragment) {
			t.Fatalf("action manifest missing %q:\n%s", fragment, actionManifest)
		}
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
		if len(ctx.Labels) != 2 || ctx.Labels[0] != "left" || ctx.Labels[1] != "right" {
			t.Fatalf("labels = %#v", ctx.Labels)
		}
		if len(ctx.Values) != 2 {
			t.Fatalf("values = %#v", ctx.Values)
		}
		leftValue, err := ctx.ValueFor("left")
		if err != nil {
			return nil, err
		}
		rightValue, err := ctx.ValueFor("right")
		if err != nil {
			return nil, err
		}
		left := leftValue.(Lexeme)
		right := rightValue.(Lexeme)
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
		SemanticActionPair: TypedPair(func(ctx PairReduction) (string, error) {
			return ctx.Left.Text + ctx.Right.Text, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if value != "ab" {
		t.Fatalf("typed map reducer value = %#v, want ab", value)
	}
	if err := (ReducerMap{}).ValidateCoverage(); err == nil || !strings.Contains(err.Error(), "pair") {
		t.Fatalf("missing reducer coverage error = %v", err)
	}
	unknownCoverage := ReducerMap{
		SemanticActionPair: TypedPair(func(ctx PairReduction) (string, error) {
			return ctx.Left.Text + ctx.Right.Text, nil
		}),
		SemanticAction(99): func(ctx Reduction) (Value, error) { return nil, nil },
	}
	if err := unknownCoverage.ValidateCoverage(); err == nil || !strings.Contains(err.Error(), "firstUnknown=99") {
		t.Fatalf("unknown reducer coverage error = %v", err)
	}
	if _, err := ParseWithReducer(tokens, ReducerMap{}); err == nil || !strings.Contains(err.Error(), "coverage") {
		t.Fatalf("parse coverage error = %v", err)
	}
	if _, err := (Reduction{Rule: 1, Action: "pair"}).ValueFor("left"); err == nil || !strings.Contains(err.Error(), "no RHS label") {
		t.Fatalf("missing label error = %v", err)
	}
	if _, err := NewPairReduction(Reduction{
		Rule:     1,
		ActionID: SemanticActionPair,
		Action:   "pair",
		Labels:   []string{"left", "right"},
		Values:   []Value{"not a lexeme", Lexeme{Text: "b"}},
	}); err == nil || !strings.Contains(err.Error(), "want semantic.Lexeme") {
		t.Fatalf("typed context mismatch error = %v", err)
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
	for _, name := range []string{"Tokens.g.cs", "Scanner.g.cs", "Parser.g.cs", "langforge.actions.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("generated C# file %s missing: %v", name, err)
		}
	}
	parserSource := readFile(t, filepath.Join(out, "Parser.g.cs"))
	for _, fragment := range []string{"record ParseResult", "class ParseException", "ParseRecovering"} {
		if !strings.Contains(parserSource, fragment) {
			t.Fatalf("generated C# parser missing %q:\n%s", fragment, parserSource)
		}
	}
	if !strings.Contains(parserSource, "Grammar rule") || !strings.Contains(parserSource, "{csharp: add}") || !strings.Contains(parserSource, "Source: "+specPath+":") {
		t.Fatalf("generated C# parser does not annotate generated tables with source grammar rules:\n%s", parserSource)
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

func findCppCompiler() (string, bool) {
	compilers := findCppCompilers()
	if len(compilers) > 0 {
		return compilers[0], true
	}
	return "", false
}

func findCppCompilers() []string {
	seen := map[string]bool{}
	var compilers []string
	for _, name := range []string{"g++", "clang++", "c++"} {
		path, err := exec.LookPath(name)
		if err == nil && !seen[path] {
			seen[path] = true
			compilers = append(compilers, path)
		}
	}
	return compilers
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

func recoverySpec(target, packageName string) string {
	return `%target ` + target + `
%package ` + packageName + `
%token Ident Number Assign Semi
%alias Ident "identifier"
%alias Number "number literal"
%group value Ident Number
%hide-expected Semi
%% lexer
IDENT = [A-Za-z_] [A-Za-z0-9_]*;
NUMBER = [0-9]+;
IDENT => token(Ident);
NUMBER => token(Number);
"=" => token(Assign);
";" => token(Semi);
[1-32]+ => skip;
%% parser
Program : Statements ;
Statements : Statement Statements | %empty ;
Statement : Ident Assign Number Semi | error Semi ;
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

func cppCalcSpec() string {
	return `%target cpp
%package LangForge::Examples::Calc::Generated
%semantic cpp mode reducer
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
S : Expr {cpp: start} ;
Expr : Expr Plus Term {cpp: add}
     | Expr Minus Term {cpp: subtract}
     | Term {cpp: pass}
     ;
Term : Term Mul Factor {cpp: multiply}
     | Term Div Factor {cpp: divide}
     | Factor {cpp: pass}
     ;
Factor : Number {cpp: number}
       | LParen Expr RParen {cpp: group}
       | Minus Factor {cpp: negate}
       ;
`
}
