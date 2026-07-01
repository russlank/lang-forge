# LangForge Architecture

Document id: `lang-forge-architecture-v1`

Status: `active`

Last updated: `2026-06-19`

Owner: `Project maintainers`

Scope: `Implementation architecture for the LangForge Go tool`

## Overview

LangForge is organized around a target-neutral compiler-tooling core:

```text
cmd/lang-forge
  CLI entry point

internal/app
  CLI orchestration, exit codes, validation, inspection, generation command wiring

internal/spec
  .lf parser plus legacy .l/.y migration parsing into a shared source model

internal/diagnostics
  Source positions, spans, diagnostics, and diagnostic lists

internal/lex
  Regex AST/parser, range algebra, alphabet partitioning, NFA construction,
  DFA subset construction, DFA minimization, longest-match tokenization

internal/parse
  Grammar model, nullable/first/follow sets, LR(0), SLR, LALR(1), IELR(1),
  canonical LR(1), action/goto table construction, conflict recording

internal/parseralgo
  Single source of truth for supported parser table algorithms and defaults

internal/codegen/golang
  Go backend: manifest, table JSON, token constants, scanner runtime,
  parser runtime, reducer-based semantic hooks

internal/codegen/csharp
  C# backend: manifest, table JSON, token enum, scanner runtime, parser
  runtime, reducer-based semantic hooks, XML documentation comments

internal/codegen/c
  C backend: manifest, table JSON, token enum/header, scanner runtime, parser
  runtime, reducer function-pointer semantic hooks
```

The intended long-term shape is:

```text
source spec -> spec model -> lex/parse tables -> target-neutral IR -> backends
```

The current codegen packages consume the same model and table objects directly.
That is acceptable for the first vertical slices; a richer `internal/ir`
package can be introduced if the backends start sharing more generation logic.

For a slower, beginner-friendly walkthrough of these stages, read
[Learning Path](learning-path.md) first, then [Compiler Pipeline](compiler-pipeline.md).

## Lexer Pipeline

1. Parse lexer definitions and rules from `.lf` or legacy `.l`.
2. Parse regex expressions into an AST.
3. Expand named definitions.
4. Reject unsupported scanner cases, including ranges outside the active
   Unicode scalar domain and rules that can match empty input.
5. Partition overlapping character sets into deterministic disjoint alphabet
   classes over sparse Unicode scalar ranges.
6. Build an NFA for each rule and combine them under one start state.
7. Convert the NFA to DFA via subset construction.
8. Select accepting rules by lowest rule index, preserving Lex-style priority.
9. Minimize the DFA while preserving accepting-rule identity.

Runtime matching uses checked UTF-8 decoding, longest match first, then rule
priority. See [Scanner Encoding Architecture](encoding.md) for details.

## Parser Pipeline

1. Parse grammar declarations and productions from `.lf` or legacy `.y`.
2. Classify terminals and nonterminals.
3. Validate undefined symbols and reject names that are both tokens and
   nonterminals.
4. Compute nullable, first, and follow sets.
5. Build LR(0), SLR, LALR(1), IELR(1), or canonical LR(1) states depending on
   the selected algorithm.
6. Build action/goto tables.
7. Record shift/reduce and reduce/reduce conflicts.

LALR(1) is the default. `%type slr`, `%type ielr`, and `%type canonical`
select the other implemented algorithms. Current validation treats conflicts
as command failure. The table still records conflict details for inspection.

For the detailed automata shape, pseudo-code, LR(1)-not-SLR example, and
algorithm-selection guidance, see [Parser Algorithms](parser-algorithms.md).

## Go Backend

`lang-forge generate --target go` writes:

- `langforge.manifest.json`: deterministic metadata for CI and auditing.
- `langforge.actions.json`: semantic actions, normalized rules, RHS labels,
  and target semantic types.
- `langforge.tables.json`: full source model and generated tables.
- `tokens.go`: token constants and string names.
- `scanner.go`: generated scanner runtime and DFA tables.
- `parser.go`: generated table-driven parser with recognizer-only and
  reducer-backed semantic APIs.

The generated Go package is reentrant and thread-safe for the common service
shape: scanner state is stored in a mutex-protected `Scanner` instance, and
parser state is local to each parse call. Generated public Go APIs include doc
comments for editor help. The parser accepts either the token
slice returned by `Tokenize` or the same slice with one trailing `TokenEOF`;
tokens after explicit EOF are rejected. `Parse` keeps the recognizer-only path,
while `ParseValue` and `ParseWithReducer` maintain a semantic value stack and
dispatch target-tagged rule actions to user reducers. Generated parsers expose
`SemanticAction` IDs, source action labels, and `ReducerMap` so reducers can
dispatch by enum-like constants while keeping readable diagnostics.
Named RHS labels and `%semantic go type` declarations additionally generate
typed action contexts and adapters when an action has one consistent
signature. `ReducerMap` coverage validation catches missing and unknown
handlers before the standard `ParseWithReducer` path parses input.
`ParseRecovering` adds grammar-directed synchronization through the reserved
`error` terminal and returns a possibly partial value, structured diagnostics,
and an accepted flag. Expected-token entries are precomputed from parser action
rows and preserve aliases, reporting groups, and exact grouped members.
`%semantic go` directives record handwritten semantic dependencies in
manifests and table JSON. When `%semantic go mode inline` is selected,
generated `parser.go` imports declared Go packages and emits a `reduceInline`
switch for target-specific action code. Generated Go files also preserve source
references: table metadata gets source comments, and inline Go snippets get
`//line` directives so compiler diagnostics can point back to `.lf`, `.l`, or
`.y` inputs.

The C#, C, and C++ runtimes implement the same recovery state machine. C#
returns `ParseResult` and throws `ParseException` from compatibility APIs; C++
returns `ParseResult` and throws `ParseError`; C exposes an allocated
`*_parse_result` released with `*_parse_result_free`. Recovery stacks and
diagnostic collections remain local to each parse call.

## C# Backend

`lang-forge generate --target csharp` writes:

- `langforge.manifest.json`: deterministic metadata for CI and auditing.
- `langforge.actions.json`: semantic actions, normalized rules, RHS labels,
  and target semantic types.
- `langforge.tables.json`: full source model and generated tables.
- `Tokens.g.cs`: token enum and grammar-name helpers.
- `Scanner.g.cs`: generated scanner runtime and DFA tables.
- `Parser.g.cs`: generated table-driven parser with recognizer-only and
  reducer-backed semantic APIs.

Generated C# output targets nullable-aware .NET code. Scanner instances
serialize access to their mutable cursor, parser state is local to each parse
call, and parser instances can be reused concurrently when the installed
reducer is also safe. C# reducer mode mirrors Go reducer mode: `{csharp: add}`
becomes a `SemanticAction.Add` enum value and a `Reduction.Action` string that
handwritten code can dispatch through `ReducerMap`.

## C Backend

`lang-forge generate --target c` writes:

- `langforge.manifest.json`: deterministic metadata for CI and auditing.
- `langforge.actions.json`: semantic actions, normalized rules, RHS labels,
  and target semantic types.
- `langforge.tables.json`: full source model and generated tables.
- `tokens.h`: token enum and token-name helper declaration.
- `scanner.h` and `scanner.c`: generated scanner runtime and DFA tables.
- `parser.h`, `parser_typed.h`, and `parser.c`: generated table-driven parser with
  recognizer-only and reducer-backed semantic APIs.

Generated C output is dependency-free C11. Scanner state is stored in a
caller-owned `*_scanner` struct, parser stacks are allocated per parse call,
and semantic state is supplied through a reducer callback plus `void *user`.
The generated API is reentrant for independent scanner/parser instances.
Sharing one scanner struct across threads requires caller synchronization.
Action labels such as `{c: add}` become target-prefixed enum values such as
`CALC_ACTION_ADD`, and shifted terminals are passed to reducers as pointers to
generated `*_lexeme` records. `parser_typed.h` adds typed reduction context
structs, required-handler validation, typed parse wrappers, and a
`*_typed_reducer_from_boxed` migration adapter. C typed handlers receive typed
arguments but return `*_value` (`void *`) so allocation and ownership remain in
handwritten code.

## C++ Backend

`lang-forge generate --target cpp` writes:

- `langforge.manifest.json`: deterministic metadata for CI and auditing.
- `langforge.actions.json`: semantic actions, normalized rules, RHS labels,
  and target semantic types.
- `langforge.tables.json`: full source model and generated tables.
- `tokens.hpp`: strongly typed token enum and token-name helper.
- `scanner.hpp` and `scanner.cpp`: generated scanner runtime and DFA tables.
- `parser.hpp`, `parser_typed.hpp`, and `parser.cpp`: generated table-driven parser with
  recognizer-only and reducer-backed semantic APIs.

Generated C++ output targets C++17. Scanner instances store a
`std::string_view` into caller-owned source text and serialize shared cursor
access with a mutex. Parser state is local to each parse call, so a parser can
be reused concurrently when the installed reducer is also safe. Action labels
such as `{cpp: add}` become `enum class SemanticAction` values such as
`SemanticAction::Add`. Parser action/goto lookup uses static sorted arrays and
binary search, while handwritten semantics normally dispatch through the
generated `ReducerMap`. `parser_typed.hpp` adds typed reduction structs,
`typed_<action>` adapter factories, `typed_reducer_map_from_boxed`, and
`ReducerMap::validate_coverage` so missing handlers fail before parsing.

## Example Project Layout

Runnable examples keep source specs and semantic code in version control while
regenerating backend output on demand:

```text
examples/go/calc
  calc.lf
  input.calc
  Makefile
  cmd/calc-demo

examples/go/datakeeper
  datakeeper.lf
  sample.dks
  Makefile
  cmd/datakeeper-demo
  AST, compiler, VM, and report code

examples/go/draw
  draw.lf
  sample.draw
  Makefile
  cmd/draw-demo
  AST, interpreter, raster renderer, and PNG output

examples/go/vehicle-report
  vehicle.lf
  sample.vehicle
  Makefile
  cmd/vehicle-report-demo
  AST and text/XML-like report output

examples/csharp/calc
  calc.lf
  input.calc
  Makefile
  Program.cs
  Generated/

examples/csharp/datakeeper
examples/csharp/draw
examples/csharp/vehicle-report
  target-specific .lf and sample input
  Makefile
  Program.cs
  Generated/

examples/c/calc
examples/c/datakeeper
examples/c/draw
examples/c/vehicle-report
  target-specific .lf and sample input
  Makefile
  main.c
  generated/
  shared helpers under examples/c/common
```

The demo commands that import generated packages are guarded by the
`langforge_generated` Go build tag. This lets `go test ./...` pass from a clean
checkout before generated output exists, while `make -C examples/... run`
generates the recognizer, builds with the tag, and executes the sample.
C# examples do not need a build tag because SDK-style .NET projects compile
generated `.g.cs` files once the Makefile has created `Generated/`.
C examples compile generated `.c` files together with handwritten `main.c`
reducers and the shared support module when a C compiler is available.
See [Generated Code And Semantics](generated-code-and-semantics.md) for the
beginner-facing explanation of this layout.

## Documentation As Architecture

LangForge treats docs, examples, tests, and generated artifacts as part of the
architecture:

- examples should show real workflows, not toy snippets only;
- tests should capture the edge case that made a behavior necessary;
- generated output should be deterministic and easy to inspect;
- public docs should explain the user-facing reason before the internal detail;
- algorithm docs should include pseudo-code close enough to the implementation
  that a reader can move from concept to code without a large jump.

This keeps the project useful both as a tool and as compiler-learning material.
