# LangForge Benchmarks

These benchmarks are optional performance examples. They are separate from
correctness tests and normal CI because benchmark numbers depend on CPU,
runtime version, compiler version, OS scheduling, power settings, and local
machine load.

Use the numbers for local trends and before/after comparisons. Do not compare
absolute Go and C# timings as if they were language rankings unless the runs
were collected under a controlled benchmark policy.

## Quick Commands

```sh
make examples-benchmarks
make examples-benchmarks-go BENCH_TIME=2s
make examples-benchmarks-go BENCH_COUNT=5 BENCH_TIME=2s
make examples-benchmarks-go BENCH_PATTERN='CalcParse'
make examples-benchmarks-csharp
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*CalcParse*'
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*'
make examples-benchmarks-report BENCH_COUNT=10 BENCH_TIME=1s
make examples-benchmarks-verbose
make examples-benchmarks-profile
```

`make examples-benchmarks` prepares generated benchmark dependencies quietly,
runs Go benchmarks, and runs the C# BenchmarkDotNet calc-parse benchmark group
when `dotnet` is available. If `.NET` is missing, the C# part is skipped with a
short message. Explicit `make examples-benchmarks-csharp` requires `.NET`.

`make examples-benchmarks-verbose` keeps LangForge validation/generation output
visible. Use it when grammar changes, generated files, or benchmark fixtures are
being debugged.

## What Is Measured

Go benchmark names use this matrix-style vocabulary:

- `BenchmarkScanner/StreamingNext`: repeatedly pulls tokens with
  `Scanner.Next`.
- `BenchmarkScanner/MaterializeAll`: tokenizes the full source with
  `Scanner.All`.
- `BenchmarkCalcParse/ParseFromSource/TypedReducer`: parses source text with a
  generated scanner feeding the parser directly, using typed reducer adapters.
- `BenchmarkCalcParse/ParsePreTokenized/TypedReducer`: parses an existing token
  slice prepared before the timed loop, using typed reducer adapters.
- `BenchmarkCalcParse/ParseFromSource/BoxedReducer`: same source parser path,
  but with a boxed reducer map.
- `BenchmarkCalcParse/ParsePreTokenized/BoxedReducer`: parser/reducer cost over
  an existing token slice with boxed reducer handlers.
- `BenchmarkDrawParse/ParseFromSource/BuildAST`: parses a large DRAW source
  through the handwritten AST-building facade.
- `BenchmarkRecoveryParse/ParseFromSource`: runs recovering parse diagnostics
  from source.
- `BenchmarkRecoveryParse/ParsePreTokenized`: runs recovering parse diagnostics
  over an existing token slice.

C# BenchmarkDotNet classes follow the same model:

- `ScannerBenchmarks.StreamingNext`
- `ScannerBenchmarks.MaterializeAll`
- `CalcParseBenchmarks.ParseFromSource_TypedReducer`
- `CalcParseBenchmarks.ParsePreTokenized_TypedReducer`
- `CalcParseBenchmarks.ParseFromSource_BoxedReducer`
- `CalcParseBenchmarks.ParsePreTokenized_BoxedReducer`
- `DrawParseBenchmarks.ParseFromSource_BuildAst`
- `RecoveryParseBenchmarks.ParseFromSource`
- `RecoveryParseBenchmarks.ParsePreTokenized`

Recognition-only parser benchmarks are intentionally left as TODOs until the
generated APIs expose a clean no-reducer/no-AST path that does not let semantic
work dominate the timing.

## Source And Token Terminology

`ParseFromSource` includes scanner/token-source work in the timed operation:

```text
source text -> generated scanner/token source -> parser -> reducer/facade
```

`ParsePreTokenized` materializes tokens before the timed loop:

```text
source text -> tokens outside benchmark
tokens -> parser -> reducer/facade inside benchmark
```

That means `ParsePreTokenized` is useful for parser/reducer comparisons, but it
does not include tokenization cost. A future `TokenizeThenParse` benchmark can
make token materialization plus parsing explicit if that scenario becomes
important.

Typed reducer benchmarks use generated typed reducer contexts. Boxed reducer
benchmarks use the compatibility shape where values are retrieved from the
generic reduction object. The boxed path is useful for migration and overhead
comparison; typed reducers are the recommended production style for real
LangForge applications.

## Go Benchmarks

The Go suite uses the standard benchmark runner:

```sh
go test -tags langforge_generated -run '^$' -bench '.' -benchmem -benchtime 1s -count 1 ./...
```

Make variables:

```make
BENCH_TIME ?= 1s
BENCH_COUNT ?= 1
BENCH_PATTERN ?= .
```

Examples:

```sh
make examples-benchmarks-go
make examples-benchmarks-go BENCH_TIME=2s
make examples-benchmarks-go BENCH_COUNT=5 BENCH_TIME=2s
make examples-benchmarks-go BENCH_PATTERN='CalcParse'
```

Use `BENCH_COUNT=5` or `BENCH_COUNT=10` before drawing conclusions. Single Go
benchmark samples are useful as smoke checks, but they are too noisy for
performance claims.

Go output fields:

- `ns/op`: nanoseconds per operation.
- `MB/s`: bytes processed per second as reported through `b.SetBytes`.
- `tokens/s`: custom token throughput metric for scanner/parser workloads.
- `lines/s`: custom DRAW source-line throughput metric.
- `B/op`: bytes allocated per operation.
- `allocs/op`: allocations per operation.

## C# BenchmarkDotNet

C# benchmarks use BenchmarkDotNet because it is the common .NET benchmarking
tool. It handles warmup, repeated measurements, statistical summaries, memory
diagnostics, generated reports, and common benchmarking mistakes better than
manual `Stopwatch` loops.

The benchmark project is:

```text
examples/benchmarks/csharp/LangForge.Examples.Benchmarks.CSharp.csproj
```

It targets `net10.0` and keeps BenchmarkDotNet only inside the optional
benchmark project, not in normal runnable examples.

Make variables:

```make
DOTNET ?= dotnet
CSHARP_BENCH_CONFIGURATION ?= Release
CSHARP_BENCH_FILTER ?= *CalcParse*
CSHARP_BENCH_ARTIFACTS ?= dist/benchmarks/csharp
```

Examples:

```sh
make examples-benchmarks-csharp
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*CalcParse*'
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*Scanner*'
make examples-benchmarks-csharp CSHARP_BENCH_FILTER='*'
```

BenchmarkDotNet output fields:

- `Mean`: arithmetic mean of measured iterations.
- `Error`: half of the confidence interval shown by BenchmarkDotNet.
- `StdDev`: standard deviation of measured iterations.
- `Gen0`, `Gen1`, `Gen2`: garbage collections per 1000 operations.
- `Allocated`: managed memory allocated per operation.

BenchmarkDotNet writes Markdown, HTML, CSV, and JSON artifacts under
`dist/benchmarks/csharp/` when run through the Make targets. Normal Make
targets save the raw BenchmarkDotNet console stream to
`dist/benchmarks/csharp-benchmarks.txt` and print the generated Markdown
tables. Prefer those generated summaries over the raw console stream when
reviewing C# numbers.

For C# before/after work, run the same `CSHARP_BENCH_FILTER` before and after a
change, keep the generated `dist/benchmarks/csharp/results` files, and compare
the Markdown/CSV/JSON summaries. Full automated C# comparison tooling is a
future improvement; BenchmarkDotNet’s exported artifacts are the first stable
step.

Optional .NET profiling tools that pair well with these benchmarks:

- `dotnet-counters`
- `dotnet-trace`
- `dotnet-gcdump`
- BenchmarkDotNet memory diagnostics, already enabled by `[MemoryDiagnoser]`

## Reports

Generate report files under `dist/benchmarks/`:

```sh
make examples-benchmarks-report BENCH_COUNT=10 BENCH_TIME=1s
```

Report outputs include:

```text
dist/benchmarks/go-benchmarks.txt
dist/benchmarks/go-benchmarks-summary.md
dist/benchmarks/csharp-benchmarks.txt
dist/benchmarks/csharp-benchmarks-summary.md
dist/benchmarks/csharp/results/
dist/benchmarks/generated-artifacts.md
dist/benchmarks/generated-artifacts.json
```

Static generated artifact metrics are intentionally not benchmarks. They are
written by `examples-benchmarks-report` as Markdown and JSON instead of being
reported as meaningless `ns/op` timing rows.

The generated artifact report includes:

- example name;
- target;
- generated byte size;
- lexer states;
- parser states;
- parser actions;
- parser gotos;
- grammar rules;
- whether recovery productions/actions are present.

## Go Before/After Comparison

Install `benchstat` only when you need comparison reports:

```sh
go install golang.org/x/perf/cmd/benchstat@latest
```

Workflow:

```sh
make examples-benchmarks-go BENCH_COUNT=10 BENCH_TIME=1s > dist/benchmarks/before.txt
# make a performance-related change
make examples-benchmarks-go BENCH_COUNT=10 BENCH_TIME=1s > dist/benchmarks/after.txt
make examples-benchmarks-compare BEFORE=dist/benchmarks/before.txt AFTER=dist/benchmarks/after.txt
```

If `benchstat` is not installed, the compare target prints the install command
and exits without failing normal benchmark runs.

## Go CPU And Memory Profiles

Generate selected Go profiles:

```sh
make examples-benchmarks-profile
```

Outputs:

```text
dist/benchmarks/profiles/go/calc-source-typed.cpu.pprof
dist/benchmarks/profiles/go/calc-source-typed.mem.pprof
dist/benchmarks/profiles/go/draw-large.cpu.pprof
dist/benchmarks/profiles/go/draw-large.mem.pprof
```

Inspect a profile with:

```sh
go tool pprof dist/benchmarks/profiles/go/calc-source-typed.cpu.pprof
```

The profile target currently covers calc parse-from-source with typed reducers
and DRAW large-source AST construction.
