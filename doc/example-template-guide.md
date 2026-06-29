# Example Template Guide

Document id: `lang-forge-example-template-guide-v1`

Last updated: 2026-06-29

Scope: `Reusable example templates, shared testdata, and generated/handwritten boundaries`

LangForge has two kinds of examples:

- demos show richer end-to-end scenarios such as calc, DataKeeper, DRAW, and
  vehicle-report;
- templates are smaller copyable starter projects for building a new language
  with a scanner, parser, reducer, AST, compiler/runtime, diagnostics, tests,
  and generated-on-demand output.

The template folders live under [examples/templates](../examples/templates).
Each target currently has a `mini-compiler` template:

```text
examples/templates/go/mini-compiler
examples/templates/csharp/mini-compiler
examples/templates/c/mini-compiler
examples/templates/cpp/mini-compiler
```

Each template accepts the same tiny source language:

```text
print 1 + 2;
print 40 + 2;
```

The `.lf` file recognizes the syntax. The handwritten code maps generated
semantic action IDs to AST construction, lowers that AST to stack-machine
instructions, runs a mock VM, and writes a report.

## Start From A Template

Run a template before editing:

```sh
make -C examples/templates/go/mini-compiler test
make -C examples/templates/csharp/mini-compiler test
make -C examples/templates/c/mini-compiler test
make -C examples/templates/cpp/mini-compiler test
```

Copy the folder for the target language you want, then rename the folder, the
package or namespace directive in the `.lf` file, the Makefile binary/project
name, and the handwritten AST/reducer/compiler names.

Use a standalone LangForge binary by overriding `LANG_FORGE`:

```sh
make -C examples/templates/go/mini-compiler LANG_FORGE=../../../../dist/lang-forge test
```

## Generated Boundary

Generated files are ignored by Git:

- Go, C, and C++ templates write `generated/`;
- C# templates write `Generated/`;
- all templates write local runtime output under `dist/`.

The `.lf` grammar and handwritten source are the source of truth. The Makefile
regenerates scanner/parser code before build and test targets.

## Reducer Helpers

Generated parsers expose a semantic action ID for each action label such as
`{go: add}`, `{csharp: add}`, `{c: add}`, or `{cpp: add}`. Those labels are not
hard-coded behavior. The handwritten reducer maps the generated action ID to
ordinary application code.

Use small typed helpers for reducer arguments:

- check that an argument exists;
- name the role, such as `left`, `right`, or `print expression`;
- cast once at the boundary;
- return domain types such as AST nodes, statement lists, or lexemes.

That pattern keeps generated APIs flexible while making handwritten semantics
readable and easy to debug.

When the grammar has RHS labels, prefer reading values by label instead of
position. The generated Go and C# APIs expose label-aware reductions; typed
contexts go one step further when `%semantic <target> type` declarations are
present.

## Spec-To-Code Checklist

When creating or reviewing an example, line up these pieces:

| Source piece | Generated piece | Handwritten piece |
|---|---|---|
| `%target` and `%package` | Target language files and package/namespace/prefix. | Imports/includes and Makefile variables that point at the generated directory. |
| `%token` and lexer rules | Token enum/const values and `Lexeme` values with source text. | Literal decoding, number parsing, identifier validation, and domain errors. |
| `%semantic <target> type Nonterminal Type` | Action manifest type metadata; Go/C# typed contexts when the action is eligible. | AST/model types and reducer return values with the same meaning. |
| `label=Symbol` in a RHS | Manifest labels; Go/C# `ValueFor` names; Go/C# typed context fields. | Reducer helper names such as `left`, `right`, `parent`, or `jobsTag`. |
| `{target: actionName}` | Generated semantic action ID/enum and manifest action entry. | Reducer map entry or switch branch that implements the behavior. |
| `%empty` alternatives | Empty reductions with no RHS values. | Explicit empty list, no-op, or optional-value result rather than accidental fallthrough. |

The examples intentionally keep domain behavior outside generated directories.
A good starter project usually has:

- a `.lf` grammar;
- a small model or AST module;
- one parser adapter that tokenizes, parses, and wires reducers;
- reducer helpers that turn generated values into domain values;
- a compiler/interpreter/report layer that consumes the AST;
- tests for valid input, scanner failures, parser failures, empty productions,
  repeated lists, and at least one semantic edge case.

Go and C# projects can declare `%semantic <target> type` entries and use
generated typed contexts. C and C++ currently use boxed reducer contexts, so
keep checked casts in one helper layer and pass descriptive names to those
helpers until their typed context parity work lands.

## Shared Testdata

Shared fixtures live under [examples/testdata](../examples/testdata). Each
family has:

- `valid/basic.*` for normal runs;
- `invalid/scanner.*` for lexical failures;
- `invalid/parser.*` for parser or semantic failures;
- `golden/report.contains` for stable report fragments.

DRAW also has a deterministic PNG signature check. The root gate is:

```sh
make examples-testdata
```

Language-specific Makefiles default to shared valid fixtures so Go, C#, C, and
C++ demos exercise the same source programs where practical.

## Shared Makefile Fragments

Reusable fragments live under [examples/mk](../examples/mk):

- `langforge.mk` owns `validate` and `generate`;
- `go.mk` owns Go build/run/test/clean behavior;
- `csharp.mk` owns .NET build/run/test/clean behavior;
- `c.mk` owns C compiler build/run/test/clean behavior;
- `cpp.mk` owns C++ compiler build/run/test/clean behavior.

A new example Makefile should mostly define variables:

```make
REPO_ROOT ?= ../../..
SPEC := my-language.lf
BIN = $(DIST_DIR)/my-language-demo
INPUT ?= ../../testdata/my-language/valid/basic.my
LOG ?= $(DIST_DIR)/my-language.log

include ../../mk/cpp.mk
```

Use recursive `=` when the value depends on shared variables such as
`$(DIST_DIR)` or `$(GENERATED_DIR)`.

## CI Gates

The root example gate runs:

```sh
make examples-cleanliness
make examples-parity
make examples-testdata
make examples-templates
make examples-test
```

`examples-cleanliness` prevents generated or build artifacts from becoming
tracked. `examples-parity` checks cross-target calc, DataKeeper, DRAW, and
vehicle-report grammar parity. `examples-testdata` checks shared fixtures and
goldens. `examples-templates` builds and tests every mini-compiler template.
`examples-test` includes all of those checks before running the richer example
projects.
