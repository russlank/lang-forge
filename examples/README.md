# Examples

LangForge examples are organized by supported target language.

- `go/calc`, `go/datakeeper`, `go/draw`, and `go/vehicle-report` are the
  full Go generated examples.
- `csharp/calc`, `csharp/datakeeper`, `csharp/draw`, and
  `csharp/vehicle-report` mirror the Go examples with generated C# output and
  handwritten C# reducers.
- `c/calc`, `c/datakeeper`, `c/draw`, and `c/vehicle-report` mirror the same
  scenarios with generated C headers/sources and handwritten C reducers. The C
  Makefiles skip compilation when no C compiler is available, but generation
  and validation still run.
- `cpp/calc`, `cpp/datakeeper`, `cpp/draw`, and `cpp/vehicle-report` mirror
  the same scenarios with generated C++17 scanner/parser output and handwritten
  reducer-map semantics. Their Makefiles skip compilation when no C++ compiler
  is available, but generation and validation still run.
- `parser-algorithms` contains source-only parser-table fixtures shared by all
  targets.
- `templates/{go,csharp,c,cpp}/mini-compiler` contains small copyable starter
  projects with AST, reducer, compiler/runtime, diagnostics, and tests.
- `testdata` contains shared valid, invalid, and golden fixtures consumed by
  the example gates.
- `mk` contains shared Makefile fragments used by demos and templates.

Generated folders, build outputs, and demo logs are ignored. Run each example's
Makefile to regenerate the target-specific scanner/parser before building.
The repository root also provides source-health checks:

```sh
make examples-cleanliness
make examples-parity
make examples-testdata
make examples-templates
```

`examples-cleanliness` fails if generated or build artifacts become tracked by
Git. `examples-parity` currently compares the calculator grammars across Go,
C#, C, and C++ after normalizing target/package/semantic directives and action
tag prefixes. `examples-testdata` runs shared fixtures and golden checks.
`examples-templates` validates the copyable mini-compiler templates.

Read [../doc/example-template-guide.md](../doc/example-template-guide.md) for
the template layout, generated/handwritten boundary, typed reducer-helper
pattern, shared testdata, and reusable Makefile fragments.

Requirements:

- Go examples need Go and `make`.
- C# examples target `net10.0` and need the .NET `10.0` SDK.
- C examples need GCC or another C11 compiler for compile/run steps. Use
  `CC=clang` or another compiler override when needed.
- C++ examples need `g++`, `clang++`, or another C++17 compiler for
  compile/run steps. Use `CXX=clang++` or another compiler override when
  needed.
