# LangForge

LangForge is a modern Go implementation of Lex/Yacc-style compiler tooling.
It is inspired by the older Pascal
[UCDT](https://github.com/russlank/UCDT) project, but the new design uses a
target-neutral core and emits modern generated code.

[![Latest Release](https://img.shields.io/github/v/release/russlank/lang-forge?display_name=tag&sort=semver)](https://github.com/russlank/lang-forge/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-%23FFDD00.svg?&style=flat&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/russlank)

Current implementation status:

- combined `.lf` specification parsing;
- legacy split `.l` + `.y` parsing for curated UCDT-derived regression
  fixtures;
- regex parsing, character-class partitioning, NFA-to-DFA construction, and
  DFA minimization;
- LR(0), SLR, LALR(1), IELR(1), and canonical LR(1) parser-table construction
  with conflict reporting;
- CLI commands: `version`, `validate`, `inspect`, `generate`;
- named RHS labels, target-specific nonterminal semantic types, and
  deterministic `langforge.actions.json` contracts across all backends;
- Go backend output for scanner/parser tables, scanner runtime, parser
  runtime with reducer-based semantic action hooks, generated semantic action
  IDs, typed reducer contexts, reducer coverage validation, reducer maps,
  token constants, and deterministic manifests;
- C# backend output for nullable-aware scanner/parser tables, thread-safe
  scanner instances, parser reducer hooks, semantic action enums, XML
  documentation comments, and deterministic manifests;
- C backend output for conventional `tokens.h`, `scanner.h`/`.c`, and
  `parser.h`/`.c` files, reentrant scanner/parser APIs, reducer function
  pointers, semantic action enums, UTF-8 checking, and deterministic manifests;
- C++ backend output for conventional `tokens.hpp`, `scanner.hpp`/`.cpp`, and
  `parser.hpp`/`.cpp` files, thread-safe scanner instances, table-driven parser
  APIs, semantic action enums, reducer maps, UTF-8 checking, and deterministic
  manifests;
- validation for empty-matching lexer rules, token/nonterminal name collisions,
  parser conflicts, invalid Unicode scalar ranges, and unsupported scanner
  settings;
- grammar-directed parser recovery with reserved `error` productions,
  expected-token aliases/groups, structured diagnostics, and cross-target
  recovery APIs;
- language-grouped examples under `examples/go`, `examples/csharp`,
  `examples/c`, and `examples/cpp`;
- copyable mini-compiler templates under `examples/templates` for Go, C#, C,
  and C++;
- runnable calc, DataKeeper, DRAW, and vehicle-report examples for Go, C#,
  C, and C++;
- Go examples with generated parser reduction hooks, AST construction,
  stack-machine lowering, PNG rendering, and XML-like report output;
- C# examples with generated `.g.cs` scanner/parser output, .NET 10 builds,
  reducer-backed semantic handling, and console/log reports;
- C examples with generated C headers/sources, handwritten reducers, a shared
  support module, console/log reports, and a full DRAW PNG renderer;
- C++ examples with generated C++17 scanner/parser output and reducer-map
  semantic dispatch.

## Requirements

The core tool needs Go `1.26.4` or a compatible newer toolchain plus `make`.
The full example and CI suite also needs the .NET `10.0` SDK for C# examples,
GCC or another C11 compiler for C examples and Go race tests, and a C++17
compiler for C++ examples.

See [Requirements](doc/requirements.md) for the complete toolchain matrix and
target-specific notes.

## Quick Start

```sh
/usr/local/go/bin/go test ./...
/usr/local/go/bin/go run ./cmd/lang-forge validate --spec examples/go/calc/calc.lf
/usr/local/go/bin/go run ./cmd/lang-forge inspect --spec examples/go/calc/calc.lf --format text
make -C examples/go/calc run
make -C examples/go/datakeeper run
make -C examples/go/draw run
make -C examples/go/vehicle-report run
make -C examples/csharp/calc run
make -C examples/csharp/datakeeper run
make -C examples/csharp/draw run
make -C examples/csharp/vehicle-report run
make -C examples/c/calc run
make -C examples/c/datakeeper run
make -C examples/c/draw run
make -C examples/c/vehicle-report run
make -C examples/cpp/calc run
make -C examples/cpp/datakeeper run
make -C examples/cpp/draw run
make -C examples/cpp/vehicle-report run
```

If `go` is on your `PATH`, the same commands work with `go` instead of
`/usr/local/go/bin/go`. The included `Makefile` uses `/usr/local/go/bin/go` by
default because that is the toolchain location in the current workspace.

The example Makefiles run LangForge from source with
`go run ../../../cmd/lang-forge`. After building a standalone utility with
`make build`, the same examples can use it with:

```sh
make -C examples/go/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/go/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/go/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/go/vehicle-report LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/csharp/vehicle-report LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/c/vehicle-report LANG_FORGE=../../../dist/lang-forge run
make -C examples/cpp/calc LANG_FORGE=../../../dist/lang-forge run
make -C examples/cpp/datakeeper LANG_FORGE=../../../dist/lang-forge run
make -C examples/cpp/draw LANG_FORGE=../../../dist/lang-forge run
make -C examples/cpp/vehicle-report LANG_FORGE=../../../dist/lang-forge run
```

If you do not want to install a binary, the Docker image can be used as the
LangForge command:

```sh
make docker-build
docker run --rm -v "$PWD:/workspace:ro" -w /workspace lang-forge:dev \
  validate --spec examples/go/calc/calc.lf
```

Generated example output is intentionally ignored. Use these commands to return
the example directories to source-only form:

```sh
make -C examples/go/calc clean
make -C examples/go/datakeeper clean
make -C examples/go/draw clean
make -C examples/go/vehicle-report clean
make -C examples/csharp/calc clean
make -C examples/csharp/datakeeper clean
make -C examples/csharp/draw clean
make -C examples/csharp/vehicle-report clean
make -C examples/c/calc clean
make -C examples/c/datakeeper clean
make -C examples/c/draw clean
make -C examples/c/vehicle-report clean
make -C examples/cpp/calc clean
make -C examples/cpp/datakeeper clean
make -C examples/cpp/draw clean
make -C examples/cpp/vehicle-report clean
```

Build, CI, release, and Docker targets are available through the root
Makefile:

```sh
make ci
make fuzz-smoke
make golden-stability
make examples-testdata
make examples-templates
make dist VERSION=0.1.0
make docker-build
make docker-smoke
```

## Documentation

- [Learning path](doc/learning-path.md)
- [Requirements](doc/requirements.md)
- [Compiler pipeline](doc/compiler-pipeline.md)
- [Glossary](doc/glossary.md)
- [Architecture](doc/architecture.md)
- [Tool improvement roadmap](doc/tool-improvement-roadmap.md)
- [Build, pipeline, and Docker](doc/build-release.md)
- [Scanner encoding architecture](doc/encoding.md)
- [Usage](doc/usage.md)
- [Invocation and layout patterns](doc/invocation-and-layouts.md)
- [Specification format](doc/specification.md)
- [Generated code and semantics](doc/generated-code-and-semantics.md)
- [Parser algorithms](doc/parser-algorithms.md)
- [Parser error recovery](doc/parser-error-recovery.md)
- [Examples](doc/examples.md)
- [Example Template Guide](doc/example-template-guide.md)
- [UCDT legacy inspiration](doc/ucdt-legacy-inspiration.md)

## Agent Skills

Reusable Codex skills for LangForge live under [skills](skills):

- `langforge-spec-authoring` for `.lf` and legacy `.l`/`.y` grammar work.
- `langforge-example-runner` for generated example projects and demo runs.
- `langforge-project-steward` for reviews, hardening, and project-memory
  updates when private notes are present.

## Current Limits

- LALR(1) is the default parser algorithm. SLR, IELR(1), and canonical LR(1)
  can be selected with `%type slr`, `%type ielr`, or `%type canonical`.
- Scanners default to checked UTF-8 and sparse Unicode scalar ranges for the
  in-process engine plus generated Go, C#, C, and C++ output. Additional source
  encodings remain planned. See
  [Scanner encoding architecture](doc/encoding.md).
- Generated Go, C#, C, and C++ parsers accept visible tokens from `Tokenize`
  and optionally one trailing explicit EOF token. Target-tagged parser actions are
  exposed through reducer callbacks with generated action IDs/enums and
  reducer-map helpers where the target has that convenience layer. Specs can
  also opt into Go inline action mode with target-tagged semantic imports for
  advanced handwritten-library integration. Named RHS labels, target-specific
  nonterminal types, deterministic action manifests, generated Go and C# typed
  reducer contexts, and Go/C# reducer-map coverage validation are implemented.
  Equivalent typed context APIs for C and C++, debug tracing, and optional AST
  helper generation remain planned. See
  [Generated code and semantics](doc/generated-code-and-semantics.md) for a
  beginner-friendly explanation of reducer labels, generated directories, and
  Go build tags used by the runnable examples.

## License

LangForge is released under the [MIT License](LICENSE).
