# LangForge Benchmarks

These benchmarks are optional performance examples. They are intentionally
separate from correctness tests and normal CI because benchmark numbers depend
heavily on CPU, compiler, runtime version, OS scheduler, and local power
settings.

The first implemented benchmark suite is Go-based:

```sh
make examples-benchmarks
make -C examples/benchmarks/go bench
```

The Go suite uses the standard `testing` benchmark runner with `-benchmem`.
It regenerates the Go calc, DRAW, and parser-recovery examples before running,
then measures:

- generated scanner throughput with `Scanner.Next` and `Scanner.All`;
- calc-large source parsing versus token-slice parsing;
- typed reducer dispatch versus boxed reducer dispatch;
- DRAW large-source parsing through the handwritten facade;
- recovery parsing from source versus a token slice;
- generated artifact/table size metadata as custom benchmark metrics.

Use a shorter smoke run while developing benchmark code:

```sh
make -C examples/benchmarks/go smoke
```

Use a longer, more stable run when collecting numbers:

```sh
make -C examples/benchmarks/go BENCHTIME=5s COUNT=5 bench
```

The benchmark output is evidence for relative tradeoffs, not a universal
promise. Compare runs on the same machine, with the same Go version, compiler
flags, generated code, and workload size.

## Other Targets

C#, C, and C++ generated parsers already expose the same source/token-slice
choices as the examples. This directory keeps their benchmark harnesses as a
tracked follow-up rather than introducing heavyweight dependencies into the
main example gate.

Suggested target-specific approaches:

- C#: add a deterministic `Stopwatch` harness first; use BenchmarkDotNet later
  only if the added package dependency is accepted for examples.
- C: add a small `clock_gettime` or `timespec_get` harness around
  `*_parse_value_source_typed`, `*_parse_value`, and `*_parse_recovering_source`.
- C++: add a `std::chrono::steady_clock` harness around
  `parse_value(scanner, reducers)`, `parse_value(tokens, reducers)`,
  direct typed reducers, boxed reducers, and boxed-to-typed adapters.

Until those harnesses exist, use the generated examples manually for rough
local comparisons:

```sh
make -C examples/c/calc build
time examples/c/calc/dist/calc-c-demo examples/c/calc/input.calc --log /tmp/calc-c.log
time examples/c/calc/dist/calc-c-demo examples/c/calc/input.calc --boxed --log /tmp/calc-c-boxed.log

make -C examples/cpp/calc build
time examples/cpp/calc/dist/calc-cpp-demo examples/cpp/calc/input.calc --log /tmp/calc-cpp.log
time examples/cpp/calc/dist/calc-cpp-demo examples/cpp/calc/input.calc --boxed --log /tmp/calc-cpp-boxed.log
time examples/cpp/calc/dist/calc-cpp-demo examples/cpp/calc/input.calc --boxed-typed --log /tmp/calc-cpp-boxed-typed.log
```

Those commands are deliberately documented as approximate. Use generated
benchmark harnesses before publishing target-to-target performance claims.
