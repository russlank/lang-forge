# LangForge Example Projects

Use this reference when adding, running, or troubleshooting example projects.

## Standard Layout

```text
examples/<target>/<name>/
  Makefile
  README.md
  <name>.lf
  sample/input files
  cmd/<name>-demo/   # Go examples
  generated/   # generated, ignored
  Generated/   # generated, ignored for C#
  dist/        # binaries, logs, rendered output, ignored
```

Generated-dependent Go files should start with:

```go
//go:build langforge_generated
```

Command packages should also have a non-generated placeholder `main` guarded by
`//go:build !langforge_generated` so `go build ./...` works before generation.

Generated output conventions are target-specific:

- Go writes `.go` files under `generated/`.
- C# writes `*.g.cs` files under `Generated/`.
- C writes conventional `.h`/`.c` files under `generated/`.
- C++ writes conventional `.hpp`/`.cpp` files under `generated/`.
- Every reducer-based target can write `langforge.actions.json` as the semantic
  action manifest.

## Makefile Contract

Example Makefiles should expose:

- `validate`: `$(LANG_FORGE) validate --spec <name>.lf`
- `generate`: validate and write target output under `generated` or `Generated`
- `build`: generate and build the demo
- `run`: build and run sample input, writing report/output under `dist`
- `test`: generate and run target-appropriate assertions
- `clean`: remove `generated` and `dist`

Default variables:

```make
GO ?= go
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
LANG_FORGE_VERBOSITY ?= 1
LF_TARGET ?= go
GENERATED_DIR := generated
DIST_DIR := dist
```

The shared example Makefile fragment passes `--verbosity
$(LANG_FORGE_VERBOSITY)` to `validate` and `generate`. Keep the default at `1`
so manual runs show major LangForge stages. Use `LANG_FORGE_VERBOSITY=0` for
quiet source-clean checks, `2` while debugging grammar/token/action decisions,
and `3` only for small table-tracing sessions.

Use `LF_TARGET` for the LangForge generation target in shared Makefile
fragments, optionally overridden by `LANGFORGE_TARGET`. Avoid generic `TARGET`;
it is common in user shells and CI environments and must not affect grammar
generation accidentally.

C# examples also remove `bin/` and `obj/` in `clean`. Do not leave generated
or build output behind after a validation run unless the user asked to inspect
it.

## Existing Examples

- `examples/go/{calc,datakeeper,draw,parser-recovery,vehicle-report}`: full
  Go generated examples. Go builds use the `langforge_generated` tag.
- `examples/csharp/{calc,datakeeper,draw,parser-recovery,vehicle-report}`:
  generated C# output under `Generated/` and handwritten C# reducers/facades.
- `examples/c/{calc,datakeeper,draw,parser-recovery,vehicle-report}`:
  generated C output with native compiler skip behavior through `CC`.
- `examples/cpp/{calc,datakeeper,draw,parser-recovery,vehicle-report}`:
  generated C++17 output with native compiler skip behavior through `CXX`. The
  DRAW examples render PNG output.
- `examples/parser-algorithms`: source-only parser-table fixtures.
- `examples/benchmarks`: optional Go and C# performance examples. Go uses
  standard `go test -bench` plus Markdown summaries; C# uses BenchmarkDotNet.
- `examples/templates/{go,csharp,c,cpp}/mini-compiler`: copyable starter
  projects with `.lf`, handwritten reducer, AST/model, compiler/runtime,
  diagnostics, tests, and generated-on-demand output.
- `examples/templates/{go,csharp,c,cpp}/library-dsl`: reusable library-style
  starters with domain model/AST, typed reducer, parser facade, diagnostics,
  thin demo entrypoint, tests or smoke assertions, and generated-on-demand
  output.
- `examples/templates/csharp/layered-compiler`: modern C# compiler-style
  starter with `Ast/`, `Semantics/`, `Parsing/`, isolated `Generated/*.g.cs`,
  `IMiniCompilerParser`, domain `ParseResult<T>`, DI-friendly semantic policy
  injection, and a thin demo entrypoint.
- `examples/templates/cpp/layered-compiler`: modern C++17 layered compiler
  starter with public headers, isolated generated output, direct typed reducer
  handlers, move-only AST ownership, a domain parser facade, and Makefile plus
  CMake validation.

The root `Makefile` should list runnable example families in `examples-run`,
`examples-test`, and `examples-clean`. When adding an example, update both the
local files and the root targets.

## Reducer Boundary

Modern examples should make the grammar-to-code contract obvious:

- keep the public handwritten boundary consistent with
  `doc/handwritten-integration-guide.md`;
- `.lf` contains `%semantic <target> type` declarations where useful;
- RHS labels such as `left=Expr` or `target=FigureReference` match reducer
  helper names;
- `{target: action}` labels map to generated action IDs/enums;
- `langforge.actions.json` records the cross-target action manifest;
- examples should use generated typed reducer contexts/adapters when eligible;
- boxed reducer paths should be explicit boxed-path coverage and keep any
  remaining casts behind descriptive helper names.
- reusable/parser-facade examples should prefer source parsing from generated
  scanners, and use reader/stream-backed scanners where the target has a
  natural abstraction: Go `NewReaderScanner`, C# `Scanner.FromTextReader`
  or `Scanner.FromStream`, C `*_stream_scanner`, and C++ `InputStreamScanner`.
- parser-recovery examples should use generated recovery result APIs and assert
  accepted/diagnostic/discarded-token behavior rather than only checking for a
  returned parse error.

DRAW, DataKeeper, and vehicle-report are the best examples for this pattern.
Calc is the compact cross-target example for reader/stream-backed scanner input
and token-list inspection.

## Benchmark Variants

Benchmarks are optional examples, not correctness gates. Use:

```sh
make examples-benchmarks
make examples-benchmarks-go BENCH_PATTERN='CalcParse'
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*CalcParse*'
make examples-benchmarks-report BENCH_COUNT=5 BENCH_TIME=2s
```

The Go benchmark report writes `dist/benchmarks/go-benchmarks.txt` and
`dist/benchmarks/go-benchmarks-summary.md`. C# BenchmarkDotNet artifacts and
LangForge summaries live under `dist/benchmarks/csharp/` and
`dist/benchmarks/csharp-benchmarks-summary.md`. Do not commit those reports
unless a task explicitly asks for baseline artifacts.

## Bootstrap Template Direction

The current templates are source directories, not generated by the CLI yet.
Future `lang-forge init` work should use them as the source of truth for
bootstrap projects:

```sh
lang-forge init mini-compiler --target go --out ./my-dsl
lang-forge init calc --target csharp --out ./calc-demo
```

Until that command exists, copy one `examples/templates/<target>/mini-compiler`
or `examples/templates/<target>/library-dsl` folder and rename the
package/namespace, Makefile variables, project file, and README by hand. For a
larger C# or C++ compiler-style project, copy the corresponding
`examples/templates/<target>/layered-compiler` folder and rename the domain
namespace plus the generated namespace in `grammar.lf`.

## Artifact Policy

- Keep `examples/**/generated/`, `examples/**/Generated/`, and
  `examples/**/dist/` ignored. C# `bin/` and `obj/` output must also stay
  ignored.
- Do not commit regenerated recognizers unless the task explicitly calls for a
  golden fixture or bootstrapping artifact.
- Before final status, prefer `make -C examples/<target>/<name> clean` and rerun
  source-only root checks.
- Mention useful outputs in the final answer, such as `dist/*.log` or rendered
  PNG paths, even when those outputs are ignored.
