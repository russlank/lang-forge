# Examples

LangForge examples are organized by supported target language.

- `go/calc`, `go/datakeeper`, `go/draw`, `go/parser-recovery`, and
  `go/vehicle-report` are the full Go generated examples.
- `csharp/calc`, `csharp/datakeeper`, `csharp/draw`,
  `csharp/parser-recovery`, and `csharp/vehicle-report` mirror the Go examples
  with generated C# output and handwritten C# reducers.
- `c/calc`, `c/datakeeper`, `c/draw`, `c/parser-recovery`, and
  `c/vehicle-report` mirror the same scenarios with generated C
  headers/sources and handwritten C reducers. The C Makefiles skip compilation
  when no C compiler is available, but generation and validation still run.
- `cpp/calc`, `cpp/datakeeper`, `cpp/draw`, `cpp/parser-recovery`, and
  `cpp/vehicle-report` mirror the same scenarios with generated C++17
  scanner/parser output and handwritten reducer-map semantics. Their Makefiles
  skip compilation when no C++ compiler is available, but generation and
  validation still run.
- `parser-algorithms` contains source-only parser-table fixtures shared by all
  targets.
- `templates/{go,csharp,c,cpp}/mini-compiler` contains small copyable
  command-line compiler starters with AST, named RHS labels, reducer,
  compiler/runtime, diagnostics, and tests.
- `templates/{go,csharp,c,cpp}/library-dsl` contains reusable library-style
  starters with domain models, typed reducers, parser facades, diagnostics,
  thin demo entrypoints, and smoke tests.
- `templates/csharp/layered-compiler` and `templates/cpp/layered-compiler`
  contain larger compiler-style starters for applications that want stronger
  facade, ownership, diagnostics, and build-system structure from day one.
- `benchmarks` contains optional Go and C# performance examples for scanner
  throughput, source parsing versus pre-tokenized parsing, reducer dispatch
  overhead, recovery overhead, generated artifact reports, and Go profiles.
  These are not part of normal CI.
- `testdata` contains shared valid, invalid, and golden fixtures consumed by
  the example gates.
- `mk` contains shared Makefile fragments used by demos and templates.

Generated folders, build outputs, and demo logs are ignored. Run each example's
Makefile to regenerate the target-specific scanner/parser before building.
Production paths in the examples prefer scanner/token-source parsing, where
the parser pulls tokens lazily from the generated scanner. Token collection
APIs are still demonstrated where they are useful for debugging, tests, or
token inspection.
The calc examples exercise source-based value parsing, including scanner,
syntax, and reducer failures. The parser-recovery examples exercise
source-based recovery diagnostics and accepted/partial-result handling.

For reusable code, keep generated packages, namespaces, and headers behind a
handwritten facade. The target guides document the practical ownership rules:
Go and C# use ordinary managed values/results, C uses explicit init/free and
semantic-value ownership, and C++ uses RAII/domain result types with generated
boxing hidden behind typed adapters where practical.
The repository root also provides source-health checks:

```sh
make examples-cleanliness
make examples-parity
make examples-testdata
make examples-templates
make examples-benchmarks
make examples-benchmarks-report
```

`examples-cleanliness` fails if generated or build artifacts become tracked by
Git. `examples-parity` first compares the calc, DataKeeper, DRAW, and
vehicle-report grammars across Go, C#, C, and C++ after normalizing
target/package/semantic directives and target-specific action labels. It then
generates in-memory `langforge.actions.json` contracts for calc, DataKeeper,
DRAW, vehicle-report, parser-recovery, mini-compiler templates, and
library-dsl templates to catch semantic action, RHS-label, typed-context, and
recovery-reporting drift.
`examples-testdata` runs shared fixtures and golden checks.
`examples-templates` validates the maintained copyable templates.
`examples-benchmarks` runs optional benchmark examples quietly by default and is
intentionally not part of `examples-test`. The default path is quick mode:
compact Go Markdown output plus BenchmarkDotNet ShortRun for the default C#
calc-parse filter. Use `examples-benchmarks-verbose`,
`examples-benchmarks-report`, and `examples-benchmarks-profile` when generation
logs, persistent reports, or Go profiles are needed. Use `BENCH_COUNT=5` or
`10` for steadier Go conclusions and `CSHARP_BENCH_JOB=medium` or `default`
for steadier C# conclusions.

Intentional action-contract differences must be documented in
`manifest-parity.allowlist.json` with `family`, `target`, `path`, and
`reason`. Prefer changing the examples back into parity unless the difference
is genuinely target-specific.

Read [../doc/example-template-guide.md](../doc/example-template-guide.md) for
the template layouts, generated/handwritten boundary, named RHS labels, typed
reducer-helper pattern, parser facade shape, shared testdata, and reusable
Makefile fragments.

Requirements:

- Go examples need Go and `make`.
- C# examples target `net10.0` and need the .NET `10.0` SDK.
- C examples need GCC or another C11 compiler for compile/run steps. Use
  `CC=clang` or another compiler override when needed.
- C++ examples need `g++`, `clang++`, or another C++17 compiler for
  compile/run steps. Use `CXX=clang++` or another compiler override when
  needed.
