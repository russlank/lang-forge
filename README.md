# LangForge

[![Latest Release](https://img.shields.io/github/v/release/russlank/lang-forge?label=release)](https://github.com/russlank/lang-forge/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-%23FFDD00.svg?&style=flat&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/russlank)
![Target: Go](https://img.shields.io/badge/target-Go-00ADD8?logo=go&logoColor=white)
![Target: C#](https://img.shields.io/badge/target-C%23-512BD4?logo=csharp&logoColor=white)
![Target: C](https://img.shields.io/badge/target-C-A8B9CC?logo=c&logoColor=111111)
![Target: C++](https://img.shields.io/badge/target-C%2B%2B-00599C?logo=cplusplus&logoColor=white)

**LangForge is a scanner/parser generator for building DSLs, validators,
transpilers, compilers, code generators, and language tooling from one readable
`.lf` grammar file.**

Write the scanner and parser once, then generate target-native code for:

- Go
- C#
- C
- C++

LangForge is designed around typed reducer APIs, multi-target generation,
parser recovery, expected-token diagnostics, and a clean separation between
generated automata and handwritten application logic.

## Why LangForge?

When you build a small language or DSL, you usually need to:

1. split source text into tokens;
2. check that the token sequence follows a grammar;
3. turn recognized syntax into useful application objects;
4. report helpful syntax errors;
5. keep generated parsing code separate from handwritten business logic.

LangForge generates the scanner and parser machinery. Your code owns the
meaning: AST nodes, reducers, validation, compilation, interpretation,
rendering, reporting, or whatever else your language is meant to do.

## What Can You Build With It?

LangForge is useful for:

- small DSLs;
- configuration languages;
- validators;
- code generators;
- transpilers;
- educational compilers;
- report or query languages;
- parser experiments;
- language-tooling prototypes.

The repository includes calculator expressions, a DataKeeper-style scripting
DSL, the DRAW rendering language, vehicle-report parsing, parser recovery
demos, mini-compiler templates, reusable library-style DSL templates, a modern
C# layered compiler template, and a layered modern C++ compiler template with a
parser facade and CMake build.

## Typed Reducers Instead Of Stack Indexing

A grammar rule can label right-hand-side values:

```lf
%semantic go type Expr float64
%semantic go type Term float64

Expr
  : left=Expr Plus right=Term {go: add}
  | value=Term {go: pass}
  ;
```

LangForge uses those labels and semantic type declarations to generate typed
reducer contexts:

```go
reducers := parser.ReducerMap{
	parser.SemanticActionAdd: parser.TypedAdd(
		func(ctx parser.AddReduction) (float64, error) {
			return ctx.Left + ctx.Right, nil
		},
	),
}
```

Handwritten reducer code can use `ctx.Left` and `ctx.Right` instead of manual
positions such as `ctx.Values[0]` and `ctx.Values[2]`.

## Pipeline

```text
source text
  -> generated scanner / lexeme source
  -> generated LR parser
  -> typed reducer
  -> AST / model / command / report
  -> compiler / interpreter / renderer / validator
```

The preferred production path is pull-based and lazy: a generated scanner feeds
tokens to the generated parser as the parser asks for them. Collection-based
token APIs are still available for debugging, teaching, tests, and token-stream
inspection.

## Highlights

- One `.lf` file for scanner and parser definitions.
- Generated scanner/parser code for Go, C#, C, and C++.
- LR parser modes: SLR(1), LALR(1), IELR(1), and canonical LR(1).
- Named grammar values such as `left=Expr` and `right=Term`.
- Typed reducer contexts/adapters instead of manual parser-stack indexing.
- Pull-based lexeme-source parsing for lazy scanner-to-parser pipelines.
- Parser error recovery with expected-token diagnostics.
- Deterministic `langforge.actions.json` action manifests.
- Copyable mini-compiler, library-style DSL, and modern C#/C++ layered templates.
- Example parity gates for cross-target grammar and semantic-contract drift.

## Quick Start

Validate a grammar:

```sh
go run ./cmd/lang-forge validate --spec examples/go/calc/calc.lf
```

Inspect parser tables:

```sh
go run ./cmd/lang-forge inspect --spec examples/go/calc/calc.lf --format text
```

Run a Go example:

```sh
make -C examples/go/calc run
```

Run a reusable library-style template:

```sh
make -C examples/templates/go/library-dsl test
```

If `go` is not on your `PATH`, pass the toolchain path as a Make override, for
example `make GO=/path/to/go build`.

## Choose Your Starting Point

| Goal | Start here |
|---|---|
| Learn the basics | [examples/go/calc](examples/go/calc) |
| Build a small compiler pipeline | [examples/templates/go/mini-compiler](examples/templates/go/mini-compiler) |
| Build a reusable DSL library | [examples/templates/go/library-dsl](examples/templates/go/library-dsl) |
| Build a layered C# compiler facade | [examples/templates/csharp/layered-compiler](examples/templates/csharp/layered-compiler) |
| Build a layered C++ compiler facade | [examples/templates/cpp/layered-compiler](examples/templates/cpp/layered-compiler) |
| See parser recovery | [examples/go/parser-recovery](examples/go/parser-recovery) |
| See a renderer-style language | [examples/go/draw](examples/go/draw) |
| Compare target languages | [examples](examples) |
| Understand automata and generated tables | [doc/automata-and-tables.md](doc/automata-and-tables.md) |
| Understand generated semantics | [doc/generated-code-and-semantics.md](doc/generated-code-and-semantics.md) |
| Understand handwritten integration | [doc/handwritten-integration-guide.md](doc/handwritten-integration-guide.md) |

## Generated Targets

| Target | Generated output | Semantic API | Notes |
|---|---|---|---|
| Go | `tokens.go`, `scanner.go`, `parser.go` | typed reducer contexts, reducer maps | primary workflow and richest examples |
| C# | `Tokens.g.cs`, `Scanner.g.cs`, `Parser.g.cs` | typed reducer contexts, action enums | nullable-aware `.g.cs` output |
| C | `tokens.h`, `scanner.h`/`.c`, `parser.h`/`.c`, `parser_typed.h` | typed reducer structs, function pointers | reentrant APIs and explicit ownership |
| C++ | `tokens.hpp`, `scanner.hpp`/`.cpp`, `parser.hpp`/`.cpp`, `parser_typed.hpp` | typed adapters and reducer maps | C++17 output |

All targets also write deterministic manifest files, including
`langforge.actions.json`, so examples and downstream projects can verify the
semantic contract produced from a grammar.

## Examples And Templates

[examples](examples) are runnable projects that show LangForge in several
language families:

- [Go examples](examples/go)
- [C# examples](examples/csharp)
- [C examples](examples/c)
- [C++ examples](examples/cpp)
- [Parser algorithm fixtures](examples/parser-algorithms)
- [Shared test data](examples/testdata)

[examples/templates](examples/templates) are copyable starting points. The
mini-compiler templates show a small front end, stack-machine lowering, and
mock execution. The library-style DSL templates hide generated parser details
behind a stable domain API, which is the recommended shape for real tools. The
C# layered compiler template shows `Ast/`, `Semantics/`, `Parsing/`, a public
`IMiniCompilerParser`, domain `ParseResult<T>`, and DI-friendly semantic
policy injection. The C++ layered compiler template goes one step further with
public headers under `include/`, generated output isolated under `generated/`,
direct typed reducers, intentional `std::unique_ptr`/`std::variant` ownership,
a domain parser facade, and CMake integration.

## Parser Algorithms

LangForge builds LR parser automata and reports conflicts with source spans
where possible. LALR(1) is the default because it is compact and familiar, but
grammars can select other algorithms:

- `%type slr` for SLR(1);
- `%type lalr` for LALR(1);
- `%type ielr` for IELR(1);
- `%type canonical` for canonical LR(1).

LR(0) item sets are part of the implementation and documentation, even when
the selected parser table uses a lookahead-aware algorithm. Start with
[Automata and driving tables](doc/automata-and-tables.md) for visual boxes and
tables, then read [Parser algorithms](doc/parser-algorithms.md) for worked
examples, automata shape, conflict behavior, and when to choose each mode.

## Error Recovery And Diagnostics

Generated parsers support grammar-directed recovery with reserved `error`
productions, synchronization terminals, expected-token aliases and groups,
partial results, and structured diagnostics. Recovery examples are available
for all generated targets:

- [Go parser recovery](examples/go/parser-recovery)
- [C# parser recovery](examples/csharp/parser-recovery)
- [C parser recovery](examples/c/parser-recovery)
- [C++ parser recovery](examples/cpp/parser-recovery)

See [Parser error recovery](doc/parser-error-recovery.md) for the grammar
patterns and generated APIs.

## Install Or Update

Install the latest release binary with `curl`:

```sh
curl -fsSL https://github.com/russlank/lang-forge/releases/latest/download/install-lang-forge.sh | sh
```

Or with `wget`:

```sh
wget -qO- https://github.com/russlank/lang-forge/releases/latest/download/install-lang-forge.sh | sh
```

The installer detects the supported OS/architecture pair, downloads the
matching release binary, verifies `SHA256SUMS`, and installs `lang-forge` to
`${PREFIX:-/usr/local}/bin`. Use a user-writable directory when you do not want
`sudo`:

```sh
curl -fsSL https://github.com/russlank/lang-forge/releases/latest/download/install-lang-forge.sh \
  | LANG_FORGE_INSTALL_DIR="$HOME/.local/bin" sh
```

Set `LANG_FORGE_REPO_URL` when installing from a fork or mirror that publishes
the same release asset names.

From a source checkout, you can also run the tool directly:

```sh
go run ./cmd/lang-forge version
```

## Requirements

The core tool needs Go `1.26.4` or a compatible newer toolchain plus `make`.
The full example and CI suite also needs the .NET `10.0` SDK for C# examples,
GCC or another C11 compiler for C examples and Go race tests, and a C++17
compiler for C++ examples.

See [Requirements](doc/requirements.md) for the complete toolchain matrix and
target-specific notes.

## Running Examples

The main demos exist in Go, C#, C, and C++:

| Example | Go | C# | C | C++ |
|---|---|---|---|---|
| Calculator | `make -C examples/go/calc run` | `make -C examples/csharp/calc run` | `make -C examples/c/calc run` | `make -C examples/cpp/calc run` |
| DataKeeper DSL | `make -C examples/go/datakeeper run` | `make -C examples/csharp/datakeeper run` | `make -C examples/c/datakeeper run` | `make -C examples/cpp/datakeeper run` |
| DRAW renderer | `make -C examples/go/draw run` | `make -C examples/csharp/draw run` | `make -C examples/c/draw run` | `make -C examples/cpp/draw run` |
| Vehicle report | `make -C examples/go/vehicle-report run` | `make -C examples/csharp/vehicle-report run` | `make -C examples/c/vehicle-report run` | `make -C examples/cpp/vehicle-report run` |
| Parser recovery | `make -C examples/go/parser-recovery run` | `make -C examples/csharp/parser-recovery run` | `make -C examples/c/parser-recovery run` | `make -C examples/cpp/parser-recovery run` |

Example Makefiles run LangForge from source by default with
`go run ../../../cmd/lang-forge`. After building a standalone utility, the same
examples can use it:

```sh
make build
make -C examples/go/calc LANG_FORGE=../../../dist/lang-forge run
```

Example Makefiles default to `LANG_FORGE_VERBOSITY=1`, so generation prints
major LangForge stages on stderr. Use `LANG_FORGE_VERBOSITY=0` for quiet runs,
or `LANG_FORGE_VERBOSITY=2`/`3` while debugging grammars and parser tables.

Generated example output is intentionally ignored. Use these commands to run
the suite and confirm the examples return to source-only form:

```sh
make examples-test
make examples-run
make examples-cleanliness
make vocabulary-check
```

`make vocabulary-check` keeps generated API names and documentation language
aligned with the glossary: Go uses explicit `FromLexemeSource` names, C# uses
overloads, C uses prefixed `_lexeme_source`/`_tokens` names, and C++ uses
overloads.

Optional benchmark examples are available separately from normal CI:

```sh
make examples-benchmarks
make examples-benchmarks-go BENCH_COUNT=5 BENCH_TIME=2s
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*CalcParse*'
make examples-benchmarks-csharp CSHARP_BENCH_JOB=medium CSHARP_BENCH_FILTER='*CalcParse*'
make examples-benchmarks-report BENCH_COUNT=10 BENCH_TIME=1s
```

The default benchmark workflow is quick mode: quiet generation, compact Go and
C# summaries, Go `-benchmem` data, and BenchmarkDotNet ShortRun for optional
C# measurements. Use `make examples-benchmarks-verbose` for generation logs,
`BENCH_COUNT=5` or `10` for steadier Go runs, and `CSHARP_BENCH_JOB=medium` or
`default` for steadier C# runs. Reports are written under `dist/benchmarks/`.
Benchmark numbers are approximate and environment-dependent, so compare runs on
the same machine and toolchain.

If you do not want to install a binary, a local Docker image can be used as the
LangForge command:

```sh
make docker-build
docker run --rm -v "$PWD:/workspace:ro" -w /workspace lang-forge:dev \
  validate --spec examples/go/calc/calc.lf
```

## Build, CI, Release, And Docker

Build, CI, release, and Docker targets are available through the root
Makefile:

```sh
make ci
make fuzz-smoke
make golden-stability
make vocabulary-check
make examples-testdata
make examples-templates
make examples-benchmarks
make examples-benchmarks-report
make dist VERSION=0.1.0-rc.1
make docker-build
make docker-smoke
```

See [Build, pipeline, and Docker](doc/build-release.md) and
[Release checklist](doc/release-checklist.md) plus
[Invocation and layout patterns](doc/invocation-and-layouts.md) for CI,
release artifacts, Docker usage, Makefile patterns, and multi-parser project
layouts.

## Documentation

- [Learning path](doc/learning-path.md)
- [Requirements](doc/requirements.md)
- [Compiler pipeline](doc/compiler-pipeline.md)
- [Glossary](doc/glossary.md)
- [Architecture](doc/architecture.md)
- [Tool improvement roadmap](doc/tool-improvement-roadmap.md)
- [Build, pipeline, and Docker](doc/build-release.md)
- [Release checklist](doc/release-checklist.md)
- [Scanner encoding architecture](doc/encoding.md)
- [Usage](doc/usage.md)
- [Invocation and layout patterns](doc/invocation-and-layouts.md)
- [Specification format](doc/specification.md)
- [Generated code and semantics](doc/generated-code-and-semantics.md)
- [Handwritten integration guide](doc/handwritten-integration-guide.md)
- [Parser algorithms](doc/parser-algorithms.md)
- [Parser error recovery](doc/parser-error-recovery.md)
- [Examples](doc/examples.md)
- [Example Template Guide](doc/example-template-guide.md)
- [UCDT reference](doc/ucdt-reference.md)

## Current Status And Limits

### Implemented

- Combined `.lf` specification parsing.
- Split `.l` plus `.y` parsing for curated UCDT-derived regression
  fixtures.
- Regex parsing, character-class partitioning, NFA-to-DFA construction, and
  DFA minimization.
- LR(0), SLR, LALR(1), IELR(1), and canonical LR(1) parser-table construction
  with conflict reporting.
- CLI commands: `version`, `validate`, `inspect`, and `generate`.
- Optional CLI verbosity for validation, generation, automata decisions, and
  parser-table traces.
- Named RHS labels, target-specific semantic type declarations, generated
  typed reducer contexts/adapters, and reducer coverage validation.
- Deterministic `langforge.actions.json`, `langforge.manifest.json`, and
  `langforge.tables.json` files.
- Example parity gates comparing grammar shape and action-manifest contracts
  across Go, C#, C, and C++.
- Generated Go, C#, C, and C++ scanner/parser backends with UTF-8 checking,
  reducer hooks, semantic action IDs/enums, and lexeme-source parsing.
- Validation for empty-matching lexer rules, token/nonterminal name collisions,
  parser conflicts, invalid Unicode scalar ranges, and unsupported scanner
  settings.
- Grammar-directed parser recovery with expected-token diagnostics and
  cross-target recovery APIs.
- Language-grouped examples and copyable templates for Go, C#, C, and C++.

### Planned

- Additional source encodings beyond checked UTF-8.
- More debug tracing and developer-facing automata explanations.
- Optional AST helper generation.
- Additional parity checks and reusable templates as the examples mature.

### Current API Notes

- LALR(1) is the default parser algorithm. SLR, IELR(1), and canonical LR(1)
  can be selected with `%type slr`, `%type ielr`, or `%type canonical`.
- Scanners default to checked UTF-8 and sparse Unicode scalar ranges for the
  in-process engine plus generated Go, C#, C, and C++ output. See
  [Scanner encoding architecture](doc/encoding.md).
- Pull-based lexeme sources are the preferred production API. Collection APIs
  such as `Tokenize`, `All`, `Parse(tokens, ...)`, and target equivalents remain
  available for tests, debugging, token reports, and simple examples.
- Specs can use reducer callbacks with generated action IDs/enums across all
  targets. Go also has an advanced inline action mode for projects that need
  target-specific semantic imports.

## UCDT Reference

LangForge acknowledges [russlank/UCDT](https://github.com/russlank/UCDT) as an
important reference for practical Lex/Yacc-style tooling and sample languages.
The current design uses a target-neutral core, generated APIs for Go, C#, C, and
C++, typed reducers, cross-target examples, and public documentation intended to
work as compiler-learning material.

## Agent Skills

Reusable Codex skills for LangForge live under [skills](skills):

- `langforge-spec-authoring` for `.lf` and split `.l`/`.y` grammar work.
- `langforge-example-runner` for generated example projects and demo runs.

## License

LangForge is released under the [MIT License](LICENSE).
