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
- `cpp/calc` demonstrates generated C++17 scanner/parser output with
  handwritten reducer-map semantics. Its Makefile skips compilation when no
  C++ compiler is available, but generation and validation still run.
- `parser-algorithms` contains source-only parser-table fixtures shared by all
  targets.

Generated folders, build outputs, and demo logs are ignored. Run each example's
Makefile to regenerate the target-specific scanner/parser before building.

Requirements:

- Go examples need Go and `make`.
- C# examples target `net10.0` and need the .NET `10.0` SDK.
- C examples need GCC or another C11 compiler for compile/run steps. Use
  `CC=clang` or another compiler override when needed.
- C++ examples need `g++`, `clang++`, or another C++17 compiler for
  compile/run steps. Use `CXX=clang++` or another compiler override when
  needed.
