# Generated Code And Semantics

Document id: `lang-forge-generated-code-and-semantics-v1`
Status: `active`
Last updated: `2026-06-25`
Owner: `Project maintainers`
Scope: `Beginner guide to generated files, semantic actions, reducers, Go build tags, and Go/C#/C/C++ output`

This page explains what LangForge generates, what the example projects still
write by hand, and how grammar action labels become real behavior.

It is written for readers who are new to Go, Lex/Yacc, or compiler tools.

## The Short Version

LangForge does three main jobs:

1. It reads a grammar file such as `calc.lf`.
2. It generates scanner and parser code under a `generated` directory.
3. It lets your own code decide what each parsed rule means.

LangForge does not know that calculator addition means `left + right`. The
grammar can label a rule as `{go: add}`, but the handwritten reducer defines
what `add` actually does.

## Scanner, Parser, And Semantics In Plain Words

A scanner, sometimes called a lexer, turns text into tokens:

```text
1+2
  -> Number("1"), Plus("+"), Number("2")
```

A parser checks whether the token sequence follows the grammar:

```text
Expr : Expr Plus Term
     | Term
     ;
```

A semantic layer decides what the recognized syntax means:

```text
Expr Plus Term
  -> add the left value and the right value
```

LangForge generates the scanner and parser. The application owns the semantic
layer.

## What `{go: add}` Means

In reducer mode, an action block is a label attached to a grammar alternative.
RHS labels give semantic values stable names:

```text
%semantic go type Expr float64
%semantic go type Term float64

Expr : left=Expr Plus right=Term {go: add}
     | left=Expr Minus right=Term {go: subtract}
     ;
```

The label is copied into the generated parser table. When the parser reduces
that rule, it creates a reduction context:

```text
ActionID = SemanticActionAdd
Action   = "add"
Labels   = ["left", "", "right"]
Values   = values for Expr, Plus, and Term
```

Then the parser calls user code:

```go
value, err := parser.ParseWithReducer(tokens, parser.ReducerFunc(reduce))
```

The reducer is ordinary Go code:

```go
var reducers = parser.ReducerMap{
	parser.SemanticActionAdd: parser.TypedAdd(
		func(ctx parser.AddReduction) (float64, error) {
			return ctx.Left + ctx.Right, nil
		},
	),
	parser.SemanticActionSubtract: parser.TypedSubtract(
		func(ctx parser.SubtractReduction) (float64, error) {
			return ctx.Left - ctx.Right, nil
		},
	),
}

func reduce(ctx parser.Reduction) (parser.Value, error) {
	return reducers.Reduce(ctx)
}
```

The important point: LangForge carries the `"add"` or `"subtract"` label
through as metadata and gives it a generated action ID. The reducer writes the
arithmetic.

The string field `Reduction.Action` remains useful for logs, diagnostics, and
tools. For dispatch, generated Go code also exposes `Reduction.ActionID`,
`SemanticAction*` constants, `LookupSemanticAction`, and `ReducerMap`. C#
output exposes the same shape as `SemanticAction` enum values,
`SemanticActions.TryLookupSemanticAction`, and `ReducerMap`. C output exposes a
target-prefixed enum such as `CALC_ACTION_ADD` and a reducer function pointer.
That is the preferred path for larger reducers because it avoids repeated
string comparisons and gives compiler-checked action names.

The same idea is target-neutral. The `.lf` file contains portable action
labels, while each backend exposes the labels in the idiom of its target
language.

## Typed Contexts And Coverage

For Go, an action receives a generated typed context when its result
nonterminal and labeled nonterminals have `%semantic go type` declarations,
labeled terminals use the generated `Lexeme` type, and every rule sharing the
action label has the same named field and result types.

The generated adapter performs focused runtime conversion once at the parser
boundary. Handwritten reducer logic then uses ordinary typed fields. The boxed
API remains available for gradual migration:

```go
left, err := ctx.ValueFor("left")
```

Generated `ReducerMap` values expose `ValidateCoverage`. The standard
`ParseWithReducer` entry point calls it automatically, so missing handlers fail
before input-dependent parser execution can hide the gap.

Every backend writes `langforge.actions.json`. It records actions, normalized
rules, RHS positions, optional labels, semantic types, and why an action could
not receive one consistent typed context. This is the stable contract for
tests, tooling, and future target-specific typed adapters.

For larger Go examples, put AST/model types in a small dependency-only package
when generated typed contexts need to reference them. DRAW, DataKeeper, and
vehicle-report use this pattern so `generated/` can import the model package
without importing the public package that already imports `generated/`.

## Why Reducer Mode Is The Default

Reducer mode keeps generated code separate from application behavior.

That separation is useful because:

- generated files can be deleted and recreated at any time;
- semantic code is normal source code with normal tests;
- Go, C#, C, and C++ backends can share the same grammar model;
- generated code avoids import cycles with application packages;
- beginners can inspect parsing separately from evaluation or execution.

In this mode, action names should usually be short labels:

```text
{go: number}
{go: add}
{go: program.withParameters}
{go: value.string}
```

The label names are not special to LangForge. They are just strings.
Generated backends can still turn those strings into native constants for
efficient reducer dispatch.

## Inline Mode

Inline mode is an explicit escape hatch for Go generation:

```text
%semantic go mode inline
%semantic go import sem "example.com/project/semantics"

Expr : Expr Plus Term {go:
    return sem.Add(ctx.Values[0], ctx.Values[2])
}
```

In inline mode, the Go action block is emitted into generated `parser.go`.
That means the generated file imports declared semantic packages and contains
target-specific Go code.

Inline mode is useful for compact experiments or tightly coupled generated
code, but reducer mode is better for the main examples because it is easier to
regenerate, test, and port to other targets.

## What `%semantic go import` Means

A spec can declare a handwritten semantic dependency:

```text
%semantic go import calcsem "example.com/project/calc/semantics"
```

In reducer mode, this is normally metadata. LangForge records the dependency
in generated metadata and exposes it from the parser package, but it does not
automatically call that package. If a `%semantic go type` uses the import alias,
the generated parser imports the package because its typed context references
that type. The application still wires reducer behavior explicitly:

```go
value, err := calc.ParseWithReducer(tokens, calc.ReducerFunc(calcsem.Reduce))
```

In inline mode, the declared import is emitted into generated `parser.go`
because the generated inline action code may call that package directly.

## What Is Generated

For Go, LangForge writes `tokens.go`, `scanner.go`, `parser.go`,
`langforge.actions.json`, `langforge.manifest.json`, and
`langforge.tables.json`.

For C#, LangForge writes `Tokens.g.cs`, `Scanner.g.cs`, `Parser.g.cs`,
`langforge.actions.json`, `langforge.manifest.json`, and
`langforge.tables.json`.

For C, LangForge writes `tokens.h`, `scanner.h`, `scanner.c`, `parser.h`,
`parser.c`, `langforge.actions.json`, `langforge.manifest.json`, and
`langforge.tables.json`.

Generated parser runtimes keep parse state local to each parse call. Go and C#
scanner instances serialize their mutable cursor so a shared scanner can be
used safely, although most applications should still create one scanner per
input. C scanners keep cursor state in a caller-owned struct. Independent C
scanner structs are reentrant and suitable for threaded programs; sharing the
same scanner struct requires caller synchronization.

In the runnable examples, LangForge writes only the local `generated`
directory for Go output:

```text
examples/go/calc/generated/
examples/go/datakeeper/generated/
examples/go/draw/generated/
examples/go/vehicle-report/generated/
```

For Go output, that directory contains:

```text
tokens.go
scanner.go
parser.go
langforge.actions.json
langforge.tables.json
langforge.manifest.json
```

Those files are ignored by Git because they are reproducible output. The
example Makefiles regenerate them before building.

For C# output, LangForge writes the local `Generated` directory:

```text
examples/csharp/calc/Generated/
examples/csharp/datakeeper/Generated/
examples/csharp/draw/Generated/
examples/csharp/vehicle-report/Generated/
```

That directory contains `Tokens.g.cs`, `Scanner.g.cs`, `Parser.g.cs`,
`langforge.actions.json`, `langforge.tables.json`, and
`langforge.manifest.json`.

For C output, LangForge writes local `generated` directories:

```text
examples/c/calc/generated/
examples/c/datakeeper/generated/
examples/c/draw/generated/
examples/c/vehicle-report/generated/
```

Each C generated directory contains `tokens.h`, `scanner.h`, `scanner.c`,
`parser.h`, `parser.c`, `langforge.actions.json`, `langforge.tables.json`, and
`langforge.manifest.json`.

For C++ output, LangForge writes local `generated` directories:

```text
examples/cpp/calc/generated/
```

Each C++ generated directory contains `tokens.hpp`, `scanner.hpp`,
`scanner.cpp`, `parser.hpp`, `parser.cpp`, `langforge.actions.json`,
`langforge.tables.json`, and `langforge.manifest.json`.

Application files outside `generated` are source-owned. For example:

```text
examples/go/calc/semantics/reducer.go
examples/go/calc/cmd/calc-demo/main.go
examples/go/datakeeper/parser.go
examples/go/draw/parser_adapter.go
examples/go/vehicle-report/parser.go
examples/csharp/*/Program.cs
examples/c/*/main.c
examples/c/common/demo.c
examples/cpp/*/main.cpp
```

Those files are not generated by LangForge. They are normal project code that
imports the generated scanner/parser package.

## Organizing More Than One Generated Parser

When one program needs several small languages, give each grammar its own
source file, generated directory, and package name:

```text
grammars/query/query.lf
grammars/policy/policy.lf
internal/generated/queryparser/
internal/generated/policyparser/
internal/query/reducer.go
internal/policy/reducer.go
```

Each generated package is independent. A Go program can import several of
them, tokenize each source with the matching scanner, and call
`ParseWithReducer` with the matching reducer:

```go
queryTokens, err := queryparser.Tokenize(querySource)
queryAST, err := queryparser.ParseWithReducer(queryTokens, queryparser.ReducerFunc(query.Reduce))

policyTokens, err := policyparser.Tokenize(policySource)
policyAST, err := policyparser.ParseWithReducer(policyTokens, policyparser.ReducerFunc(policy.Reduce))
```

A C program follows the same shape with generated prefixes:

```c
calc_lexeme *tokens = NULL;
size_t count = 0;
calc_value value = NULL;
calc_error error = {{0}};

if (!calc_tokenize(source, &tokens, &count, &error)) {
    /* handle error.message */
}
if (!calc_parse_value(tokens, count, calc_reduce, &user_state, &value, &error)) {
    /* handle error.message */
}
calc_free_lexemes(tokens);
```

A C++ program follows the same shape with generated namespaces and reducer
maps:

```cpp
namespace generated = LangForge::Examples::Calc::Generated;

auto tokens = generated::tokenize(source);
generated::ReducerMap reducers{
    {generated::SemanticAction::Add, [](const generated::Reduction& ctx) {
        return std::any_cast<double>(ctx.values[0]) +
               std::any_cast<double>(ctx.values[2]);
    }},
};
auto value = generated::parse_value(tokens, reducers);
```

The generated C++ parser tables use static arrays and binary search for
action/goto lookup. Semantic labels are exposed as `enum class
SemanticAction` values; handwritten reducers normally use `ReducerMap` rather
than a long `switch`.

Keep handwritten reducers, AST nodes, compilers, interpreters, and adapters
outside generated directories. Generated directories should either be ignored
and recreated by Makefiles, as the examples do, or committed with a CI check
that fails when regeneration changes them.

See [Invocation And Layout Patterns](invocation-and-layouts.md) for
Makefile templates, Docker invocation, and larger multi-parser layouts.

For C examples, handwritten code includes generated headers with paths such as
`generated/parser.h`. That avoids checked-in compatibility stubs and keeps the
generated header as the only definition of types like `calc_lexeme`, while
still allowing IDEs to resolve the API without understanding the Makefile's
`-Igenerated` flag. Run the example `generate` target once before expecting IDE
navigation into generated C headers.

C++ examples follow the same IDE-friendly convention with
`generated/parser.hpp`, so generated types such as `Lexeme`, `SemanticAction`,
and `ReducerMap` have one definition while editors can still find them.

## Source References In Generated Code

Generated Go files include source comments that point back to the `.lf`, `.l`,
or `.y` input used to create them. For example, generated scanner and parser
tables include comments near rule metadata entries:

```go
// Source: vehicle.lf:57:1
```

Those comments are for humans reading generated code or debugging table
metadata. They do not change how the Go compiler sees generated table data.

Inline Go semantic snippets are different because they are user-written Go
code emitted into generated `parser.go`. LangForge wraps those snippets with
Go line directives:

```go
//line vehicle.lf:57:1
```

That lets Go compiler errors in inline snippets point back to the grammar
source. Reducer mode usually does not need compiler line directives because
the semantic behavior lives in normal handwritten Go files outside
`generated`.

## Why Some Go Files Use `//go:build langforge_generated`

Go build tags are conditional-compilation comments. A file that starts with:

```go
//go:build langforge_generated
```

is compiled only when the Go command receives:

```sh
go build -tags langforge_generated
go test -tags langforge_generated
```

The examples use this because generated code is intentionally not committed.
Before `make run` or `make test`, the `generated` package may not exist. Files
that import that package must therefore be excluded from ordinary root-level
builds.

The example Makefiles do this sequence:

```text
validate grammar
generate examples/.../generated
go build -tags langforge_generated
run the demo
```

Some demo command packages also include a fallback file with:

```go
//go:build !langforge_generated
```

That means "compile this file when the generated-code tag is not set." The
fallback exists only to print a helpful message instead of failing with a
missing import.

## Calc Walkthrough

Calc is the smallest end-to-end example:

```text
calc.lf
  -> lang-forge generate
  -> generated scanner/parser
  -> calc-demo main.go
  -> semantics/reducer.go
  -> printed numeric result
```

The grammar says:

```text
Expr : Expr Plus Term {go: add}
     | Expr Minus Term {go: subtract}
     ;
```

The generated parser stores:

```text
rule Expr -> Expr Plus Term has action "add"
rule Expr -> Expr Minus Term has action "subtract"
```

The handwritten reducer says:

```text
if action is "add", return left + right
if action is "subtract", return left - right
```

This is the same pattern used by the larger examples:

- DataKeeper uses named action/RHS labels and Go typed contexts to build a
  script AST, then lowers that AST to stack-machine instructions.
- DRAW uses action labels to build a drawing AST, then interprets it into a
  PNG image.
- Vehicle report uses named action/RHS labels and Go typed contexts to build a
  small AST, then renders a text/XML-like report.

## Recovery Productions And Semantic Actions

A recovery production can have an ordinary target action:

```lf
Statement : error Semi {go: recover.statement}
```

The action runs when the recovery rule reduces, just like any other grammar
action. Do not label or depend on the semantic value of `error`; it is a
control symbol rather than scanner input. Use labeled values after `error`, or
application state owned by the reducer, to construct a placeholder AST node or
record that a statement was skipped.

Recovery diagnostics are separate from semantic reducer errors. A generated
recovery API can return a partial semantic value together with syntax
diagnostics, while an error returned or thrown by handwritten reducer code
still stops parsing as an application failure.

See [Parser Error Recovery](parser-error-recovery.md) for the complete runtime
contract and target APIs.

## Mental Model

Use this rule of thumb:

```text
.lf file        describes syntax and names semantic hooks
generated/      recognizes tokens and grammar rules
reducer/adapter defines what recognized rules mean
cmd/            runs a complete demo or application
dist/           contains built binaries and logs
```

When reading an example, start with the `.lf` file, then look at the reducer or
adapter code outside `generated`. Open generated files only when you want to
study the scanner/parser tables or generated API shape.
