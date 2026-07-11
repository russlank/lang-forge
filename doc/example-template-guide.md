# Example Template Guide

Document id: `lang-forge-example-template-guide-v1`

Last updated: 2026-07-10

Scope: `Reusable example templates, shared testdata, and generated/handwritten boundaries`

LangForge has two kinds of examples:

- demos show richer end-to-end scenarios such as calc, DataKeeper, DRAW, and
  vehicle-report;
- templates are smaller copyable starter projects for building a new language
  with a scanner, parser, reducer, AST, compiler/runtime, diagnostics, tests,
  and generated-on-demand output.

The template folders live under [examples/templates](../examples/templates).
Each target currently has two cross-target template families. C# and C++ also
have larger layered compiler templates for projects that want stronger
application architecture around generated parser code:

```text
examples/templates/go/mini-compiler
examples/templates/go/library-dsl
examples/templates/csharp/mini-compiler
examples/templates/csharp/library-dsl
examples/templates/csharp/layered-compiler
examples/templates/c/mini-compiler
examples/templates/c/library-dsl
examples/templates/cpp/mini-compiler
examples/templates/cpp/library-dsl
examples/templates/cpp/layered-compiler
```

Choose `mini-compiler` when you want a compact end-to-end command-line demo
that parses, builds an AST, lowers to a tiny stack machine, and prints a
report. Choose `library-dsl` when you want the recommended starting shape for a
real application: generated parser details hidden behind a domain parser
facade, explicit diagnostics, reusable model types, and a thin demo entrypoint.
Choose `csharp/layered-compiler` when you want a compiler-style C# starter
with `Ast/`, `Semantics/`, `Parsing/`, domain `ParseResult<T>`,
`IMiniCompilerParser`, and DI-friendly semantic policy injection.
Choose `cpp/layered-compiler` when you specifically want the modern C++ version
of that idea with public headers, direct typed reducer handlers, move-only AST
ownership, source-based parsing, and CMake.

The `mini-compiler` templates accept the same tiny source language:

```text
print 1 + 2;
print 40 + 2;
```

The `.lf` file recognizes the syntax. The handwritten code maps generated
semantic action IDs to AST construction, lowers that AST to stack-machine
instructions, runs a mock VM, and writes a report.

The mini-compiler templates intentionally show the current recommended
LangForge style rather than a legacy boxed-only style. Each grammar declares
target-specific semantic types, labels meaningful RHS symbols, and lets
generated typed reducer contexts validate the reducer boundary.

The `library-dsl` templates accept configuration-like source:

```text
set retries = 3;
set title = "nightly";
set owner = admin;
enable audit;
```

They split handwritten code by responsibility:

| Target | Domain model | Semantics | Parser facade | Diagnostics | Entrypoint |
|---|---|---|---|---|---|
| Go | `model` package | reducer map in `parser` package | `parser.Parser` | `parser.FormatError` | `cmd/library-dsl` |
| C# | `Ast/` | `Semantics/ReducerFactory.cs` | `Parsing/ILibraryDslParser` and `LibraryDslParser` | `Parsing/DiagnosticFormatter.cs` | `Program.cs` |
| C | `ast.h`/`.c` | `semantics.h`/`.c` | `parser_facade.h`/`.c` | `diagnostics.h`/`.c` | `main.c` |
| C++ | `include/library_dsl/ast.hpp` | `src/semantics.*` | `ParserFacade` | `src/diagnostics.*` | `src/main.cpp` |

The library templates are intentionally small, but they teach the boundary a
larger application usually wants: call your own parser facade, receive your own
AST/domain model, and keep generated packages/namespaces/prefixes out of most
application code.

The C `library-dsl` template is also the recommended place to copy ownership
patterns from. It uses one per-parse allocator for reducer-created AST values,
destroys that allocator on scanner/syntax/reducer failure, transfers it to the
returned `dsl_document` on success, and exposes `dsl_parse_result_free` as the
single cleanup function an application calls.

The C# template exposes an interface and concrete parser but does not add a
dependency-injection package reference. That keeps the starter runnable with
only the .NET SDK. Applications can register `LibraryDslParser` as
`ILibraryDslParser` in their own host/container layer.

The C# `layered-compiler` template applies the same facade idea to a
compiler-style pipeline:

- `Ast/` contains public records such as `ProgramNode`, `PrintStatementNode`,
  `NumberExprNode`, and `AddExprNode`;
- `Semantics/ReducerFactory.cs` is the only handwritten layer that maps
  generated typed reducer contexts to AST construction;
- pure reducer maps are cached by the parser facade or reducer factory, while
  parse-owned state such as policies, allocators, and AST builders remains
  explicit;
- `Semantics/INumberLiteralPolicy.cs` shows where domain services can be
  injected without teaching generated code about a DI container;
- `Parsing/IMiniCompilerParser.cs` and `MiniCompilerParser.cs` expose the
  stable parser facade and return `ParseResult<ProgramNode>`;
- `Compilation/` lowers the AST to a tiny stack-machine program and executes
  the mock runtime;
- `Program.cs` only wires the demo command, assertions, parser, compiler, and
  report writer.

The C++ `layered-compiler` template is the recommended C++ architecture
reference when a project is larger than a tiny demo. It uses:

- `include/mini` for public domain headers;
- `src/parser.cpp` as the only handwritten file that includes generated parser
  headers;
- `src/compiler.cpp` for AST-to-stack-code lowering and execution;
- `std::unique_ptr` for owned AST nodes;
- `std::variant` for closed AST node families;
- lightweight semantic handles on the parser stack, so generated typed
  reducers remain copyable;
- `mini::Result<T>` as a C++17 stand-in for `std::expected`;
- a `CMakeLists.txt` that generates LangForge output before compiling the demo.

## Production Responsibilities

Templates are meant to be copied, so they show the responsibilities that should
stay outside generated code:

| Responsibility | Template pattern |
|---|---|
| Domain model or AST ownership | Go `model`, C# `Ast/`, C `ast.h/c`, C++ `include/.../ast.hpp` |
| Reducer behavior | Generated typed contexts mapped to ordinary handwritten functions |
| Parser public API | A facade such as `Parser`, `ILibraryDslParser`, `dsl_parse_lexeme_source`, or `ParserFacade` |
| Diagnostics policy | Generated diagnostics converted to messages meaningful to the domain |
| Cleanup | C result/free functions, C++ `unique_ptr`/RAII, Go/C# ordinary ownership |
| Runtime/compiler layer | Separate compiler, renderer, report writer, or mock VM code |

Generated scanner/parser code is an implementation detail. Keep imports of
generated packages, namespaces, and headers inside the facade or semantic
adapter where practical. This lets a larger application call stable domain
methods without depending on generated parser stack values.

The normal production path in all templates is source-based parsing:

```text
source text or reader/stream -> generated scanner/lexeme source -> parser -> reducer -> AST/domain result
```

For small examples a string overload is convenient. For reusable libraries,
prefer a facade overload that accepts the target's demand-fed input shape:
Go `io.Reader`, C# `TextReader` or `Stream`, C read callbacks, and C++
`std::istream`. Keep token-list parsing for tests, tutorials, and token
inspection. The calc demo family shows the compact cross-target reader/stream
pattern; the library templates show how to hide generated parser details
behind a domain facade.

Normal reducer failures should be returned or reported through the generated
parser API, not through process-level failures. Go reducers return `error`, C
reducers fill the generated error struct and return `NULL`, and C#/C++
reducers may throw exceptions that the handwritten facade converts into a
domain result when exception-free callers are desired. Avoid `panic`, `abort`,
and unchecked cast helpers for user-input failures.

## Start From A Template

Run a template before editing:

```sh
make -C examples/templates/go/mini-compiler test
make -C examples/templates/go/library-dsl test
make -C examples/templates/csharp/mini-compiler test
make -C examples/templates/csharp/library-dsl test
make -C examples/templates/csharp/layered-compiler test
make -C examples/templates/c/mini-compiler test
make -C examples/templates/c/library-dsl test
make -C examples/templates/cpp/mini-compiler test
make -C examples/templates/cpp/library-dsl test
make -C examples/templates/cpp/layered-compiler test
make -C examples/templates/cpp/layered-compiler cmake-test
```

Copy the folder for the target language you want, then rename the folder, the
package or namespace directive in the `.lf` file, the Makefile binary/project
name, and the handwritten AST/reducer/facade names. For an application library,
start from `library-dsl`; for a compact compiler pipeline, start from
`mini-compiler`. For a C# compiler-style library facade, start from
`csharp/layered-compiler`. For a C++ project where ownership,
generated-code isolation, and CMake matter from day one, start from
`cpp/layered-compiler`.

Use a standalone LangForge binary by overriding `LANG_FORGE`:

```sh
make -C examples/templates/go/mini-compiler LANG_FORGE=../../../../dist/lang-forge test
make -C examples/templates/go/library-dsl LANG_FORGE=../../../../dist/lang-forge test
```

## Generated Boundary

Generated files are ignored by Git:

- Go, C, and most C++ templates write `generated/`;
- the C++ `library-dsl` template writes `src/generated/` so public headers and
  generated headers are both reachable from a conventional `include`/`src`
  layout;
- the C++ `layered-compiler` template writes root-level `generated/` and keeps
  generated includes private to `src/parser.cpp`;
- C# templates write `Generated/` with `.g.cs` source files;
- all templates write local runtime output under `dist/`.

The `.lf` grammar and handwritten source are the source of truth. The Makefile
regenerates scanner/parser code before build and test targets.

## Typed Reducers

Generated parsers expose a semantic action ID for each action label such as
`{go: add}`, `{csharp: add}`, `{c: add}`, or `{cpp: add}`. Those labels are not
hard-coded behavior. The handwritten reducer maps the generated action ID to
ordinary application code.

For real projects, prefer generated typed reducer contexts:

- declare `%semantic <target> type Nonterminal TargetType`;
- label grammar values by role, such as `left=Expr`, `right=Term`,
  `expr=Expr`, and `token=Number`;
- use generated adapters such as Go `TypedAdd`, C# `TypedAdd`, C
  `mini_compiler_parse_value_lexeme_source_typed`, or C++ `typed_add`;
- keep boxed `ctx.Values[index]` access only in migration shims or debugging
  code.
- return reducer/semantic failures through the parser API. In Go, return
  `error`; in C, fill the generated error struct and return `NULL`; in C# and
  C++, throw ordinary exceptions from the parser call path. Avoid `panic`,
  `abort`, or unchecked cast helpers for normal user input failures.

For example, this grammar rule:

```lf
Expr : left=Expr Plus right=Term
         {go: add}
     ;
```

becomes a Go reducer handler with named, typed fields:

```go
minigen.SemanticActionAdd: minigen.TypedAdd(func(ctx minigen.AddReduction) (minimodel.Expr, error) {
    return minimodel.AddExpr{Left: ctx.Left, Right: ctx.Right}, nil
}),
```

The C# template uses the same idea:

```csharp
[SemanticAction.Add] = TypedAdd(ctx => new AddExpr(ctx.Left, ctx.Right)),
```

The C template receives a generated C context:

```c
static mini_compiler_value reduce_add(
    const mini_compiler_add_reduction *ctx,
    void *user,
    mini_compiler_error *error)
{
    context *state = (context *)user;
    (void)error;
    return new_add(state, ctx->left, ctx->right);
}
```

The compact C++ templates use `parser_typed.hpp` adapters:

```cpp
{mini::SemanticAction::Add, mini::typed_add([](const mini::AddReduction& ctx) -> ExprPtr {
    return add_expr(ctx.left, ctx.right);
})},
```

The layered C++ template uses the same generated typed adapters, but keeps the
reducer map inside `src/parser.cpp` and returns domain values from direct typed
handlers:

```cpp
// Grammar: Expr : left=Expr Plus right=Term {cpp: add}
{lfgen::SemanticAction::Add, lfgen::typed_add(
    [&session](const lfgen::AddReduction& ctx) -> mini::ast::ExprId {
        return session.builder.add(ctx.left, ctx.right);
    })},
```

The handwritten path does not call `std::any_cast`; any boxing needed by the
generated runtime stays behind the generated typed adapter boundary.

Every template also writes `langforge.actions.json`. Review that file when
changing a grammar: intended actions should have `"typed": true`, and field
entries should show the labels and target types you expect.

The mini-compiler templates test one reducer failure explicitly: an oversized
integer literal is lexically valid, but the `number` reducer rejects it as a
semantic error. That is the pattern to copy for domain-level failures such as
unknown symbols, invalid ranges, duplicate declarations, or unsupported target
operations.

## Library Facade Mapping

The `library-dsl` grammar has rules like:

```lf
Entry : Set name=Ident Assign value=Value Semi
          {go: entry.set}
      | Enable name=Ident Semi
          {go: entry.enable}
      ;
Value : token=Number
          {go: value.number}
      | token=String
          {go: value.string}
      | token=Ident
          {go: value.ident}
      ;
```

Each target implements the same semantic contract with target-native names:

| Grammar piece | Go | C# | C | C++ |
|---|---|---|---|---|
| `name=Ident` | `ctx.Name` | `ctx.Name` | `ctx->name` | `ctx.name` |
| `value=Value` | `ctx.Value` | `ctx.Value` | `ctx->value` | `ctx.value` |
| `{target: entry.set}` | `SemanticActionEntrySet` | `SemanticAction.EntrySet` | `LIBRARY_DSL_ACTION_ENTRY_SET` | `SemanticAction::EntrySet` |
| `{target: value.number}` | `TypedValueNumber` | `TypedValueNumber` | `value_number` handler slot | `typed_value_number` |

The parser facades then convert the generated final value into the domain root:

```text
generated scanner/source
  -> generated parser with typed reducer map
  -> model.Document / Ast.Document / dsl_document / library_dsl::Document
  -> application API result
```

Use this pattern when the parser is part of a larger application. Keep
compatibility token-list parsing only for tests, debugging token streams, or
adapters that already own a token collection.

## Spec-To-Code Checklist

When creating or reviewing an example, line up these pieces:

| Source piece | Generated piece | Handwritten piece |
|---|---|---|
| `%target` and `%package` | Target language files and package/namespace/prefix. | Imports/includes and Makefile variables that point at the generated directory. |
| `%token` and lexer rules | Token enum/const values and `Lexeme` values with source text. | Literal decoding, number parsing, identifier validation, and domain errors. |
| `%semantic <target> type Nonterminal Type` | Action manifest type metadata; typed contexts/adapters when the action is eligible. | AST/model types and reducer return values with the same meaning. |
| `label=Symbol` in a RHS | Manifest labels, `ValueFor` names, and typed context fields. | Reducer helper names such as `left`, `right`, `parent`, or `jobsTag`. |
| `{target: actionName}` | Generated semantic action ID/enum and manifest action entry. | Reducer map entry or switch branch that implements the behavior. |
| `%empty` alternatives | Empty reductions with no RHS values. | Explicit empty list, no-op, or optional-value result rather than accidental fallthrough. |

The examples intentionally keep domain behavior outside generated directories.
A good starter project usually has:

- a `.lf` grammar;
- a small model or AST module;
- one parser adapter that feeds a generated scanner/lexeme source into the
  parser and wires reducers;
- reducer helpers that turn generated values into domain values;
- a compiler/interpreter/report layer that consumes the AST;
- tests for valid input, scanner failures, parser failures, empty productions,
  repeated lists, and at least one semantic edge case.

Projects can declare `%semantic <target> type` entries and use generated typed
contexts/adapters. Keep boxed reducer paths only when they are useful for
compatibility or migration, and keep any remaining casts in one helper layer
with descriptive grammar-role names.

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
vehicle-report grammar parity, then checks `langforge.actions.json` contract
parity for those examples plus parser-recovery, mini-compiler templates, and
library-dsl templates. `examples-testdata` checks shared fixtures and goldens.
`examples-templates` builds and tests every maintained template family,
including the C++ layered compiler template.
`examples-test` includes all of those checks before running the richer example
projects.
