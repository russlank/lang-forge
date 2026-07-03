package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionParityNormalizesTargetSpecificTypeSpellings(t *testing.T) {
	dir := t.TempDir()
	goPath := writeSpec(t, dir, "calc-go.lf", `%target go
%package calc
%semantic go mode reducer
%semantic go type S float64
%semantic go type Expr float64
%semantic go type Term float64
%start S
%token Number Plus

%% lexer
NUMBER = [0-9]+;
NUMBER => token(Number);
"+" => token(Plus);
[1-32]+ => skip;

%% parser
S : value=Expr {go: start} ;
Expr : left=Expr Plus right=Term {go: add}
     | value=Term {go: pass}
     ;
Term : token=Number {go: number}
     ;
`)
	cPath := writeSpec(t, dir, "calc-c.lf", `%target c
%package calc
%semantic c mode reducer
%semantic c type S double
%semantic c type Expr double
%semantic c type Term double
%start S
%token Number Plus

%% lexer
NUMBER = [0-9]+;
NUMBER => token(Number);
"+" => token(Plus);
[1-32]+ => skip;

%% parser
S : value=Expr {c: start} ;
Expr : left=Expr Plus right=Term {c: add}
     | value=Term {c: pass}
     ;
Term : token=Number {c: number}
     ;
`)

	goContract, err := buildContract(targetSpec{Target: "go", Path: goPath})
	if err != nil {
		t.Fatal(err)
	}
	cContract, err := buildContract(targetSpec{Target: "c", Path: cPath})
	if err != nil {
		t.Fatal(err)
	}
	if diffs := diffContracts(goContract, cContract); len(diffs) != 0 {
		t.Fatalf("expected target-specific type spellings to normalize away, got %#v", diffs)
	}
}

func TestActionParityDetectsActionLabelDrift(t *testing.T) {
	dir := t.TempDir()
	goPath := writeSpec(t, dir, "go.lf", tinySpec("go", "add"))
	cppPath := writeSpec(t, dir, "cpp.lf", tinySpec("cpp", "sum"))

	goContract, err := buildContract(targetSpec{Target: "go", Path: goPath})
	if err != nil {
		t.Fatal(err)
	}
	cppContract, err := buildContract(targetSpec{Target: "cpp", Path: cppPath})
	if err != nil {
		t.Fatal(err)
	}
	diffs := diffContracts(goContract, cppContract)
	if len(diffs) == 0 {
		t.Fatal("expected action-label drift to be reported")
	}
	if !hasDiffPath(diffs, "actions[0].name") {
		t.Fatalf("expected action name diff, got %#v", diffs)
	}
}

func TestActionParityDetectsRHSLabelDrift(t *testing.T) {
	dir := t.TempDir()
	goPath := writeSpec(t, dir, "go.lf", `%target go
%package demo
%semantic go mode reducer
%semantic go type S string
%semantic go type Expr string
%start S
%token Number

%% lexer
NUMBER = [0-9]+;
NUMBER => token(Number);
[1-32]+ => skip;

%% parser
S : value=Expr {go: start} ;
Expr : token=Number {go: number} ;
`)
	csharpPath := writeSpec(t, dir, "csharp.lf", `%target csharp
%package Demo.Generated
%semantic csharp mode reducer
%semantic csharp type S string
%semantic csharp type Expr string
%start S
%token Number

%% lexer
NUMBER = [0-9]+;
NUMBER => token(Number);
[1-32]+ => skip;

%% parser
S : value=Expr {csharp: start} ;
Expr : literal=Number {csharp: number} ;
`)

	goContract, err := buildContract(targetSpec{Target: "go", Path: goPath})
	if err != nil {
		t.Fatal(err)
	}
	csharpContract, err := buildContract(targetSpec{Target: "csharp", Path: csharpPath})
	if err != nil {
		t.Fatal(err)
	}
	diffs := diffContracts(goContract, csharpContract)
	if !hasDiffPath(diffs, "actions[1].rules[0].rhs[0].label") {
		t.Fatalf("expected RHS label diff, got %#v", diffs)
	}
}

func TestActionParityDetectsRecoveryExpectedTokenDrift(t *testing.T) {
	dir := t.TempDir()
	goPath := writeSpec(t, dir, "go.lf", recoverySpec("go", "number literal"))
	cPath := writeSpec(t, dir, "c.lf", recoverySpec("c", "numeric literal"))

	goContract, err := buildContract(targetSpec{Target: "go", Path: goPath})
	if err != nil {
		t.Fatal(err)
	}
	cContract, err := buildContract(targetSpec{Target: "c", Path: cPath})
	if err != nil {
		t.Fatal(err)
	}
	diffs := diffContracts(goContract, cContract)
	if !hasDiffPath(diffs, "expected.aliases[1].label") {
		t.Fatalf("expected expected-token alias drift, got %#v", diffs)
	}
	if !goContract.Recovery.Enabled || len(goContract.Recovery.Productions) != 1 {
		t.Fatalf("expected recovery contract, got %#v", goContract.Recovery)
	}
}

func TestAllowlistSuppressesDocumentedDifference(t *testing.T) {
	diffs := []contractDiff{
		{Path: "actions[0].name", Want: `"add"`, Got: `"sum"`},
		{Path: "actions[0].typed", Want: "true", Got: "false"},
	}
	allowed := allowlist{Allow: []allowedDifference{
		{Family: "calc", Target: "cpp", Path: "actions[0].name", Reason: "intentional test difference"},
	}}
	unallowed := filterUnallowed("calc", "cpp", diffs, allowed)
	if len(unallowed) != 1 || unallowed[0].Path != "actions[0].typed" {
		t.Fatalf("unexpected unallowed differences: %#v", unallowed)
	}
}

func writeSpec(t *testing.T, dir string, name string, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func tinySpec(target string, actionLabel string) string {
	targetAction := target
	if target == "cpp" {
		targetAction = "cpp"
	}
	return `%target ` + target + `
%package demo
%semantic ` + target + ` mode reducer
%semantic ` + target + ` type S string
%start S
%token Number

%% lexer
NUMBER = [0-9]+;
NUMBER => token(Number);
[1-32]+ => skip;

%% parser
S : token=Number {` + targetAction + `: ` + actionLabel + `} ;
`
}

func recoverySpec(target string, numberAlias string) string {
	return `%target ` + target + `
%package recovery
%token Ident Number Assign Semi
%alias Ident "identifier"
%alias Number "` + numberAlias + `"
%group value Ident Number
%hide-expected Semi
%start Program

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
Statements : Statement Statements
           | %empty
           ;
Statement : Ident Assign Number Semi
          | error Semi
          ;
`
}

func hasDiffPath(diffs []contractDiff, suffix string) bool {
	for _, diff := range diffs {
		if strings.HasSuffix(diff.Path, suffix) || diff.Path == suffix {
			return true
		}
	}
	return false
}
