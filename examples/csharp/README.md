# C# Examples

The C# examples use `lang-forge generate --target csharp` and build with the
local .NET SDK. They target `net10.0`, so the .NET `10.0` SDK is required for
`run` and `test` targets.

Run one example:

```sh
make -C examples/csharp/calc run
make -C examples/csharp/datakeeper run
make -C examples/csharp/draw run
make -C examples/csharp/vehicle-report run
```

Run generated-code checks:

```sh
make -C examples/csharp/calc test
make -C examples/csharp/datakeeper test
make -C examples/csharp/draw test
make -C examples/csharp/vehicle-report test
```

Generated scanner/parser files are written to `Generated/` with `.g.cs`
filenames and ignored by Git. Each project keeps handwritten reducer and demo
code outside `Generated/`.

The Makefiles include shared fragments from `examples/mk` and default to
shared valid fixtures under `examples/testdata`. For a smaller copyable starter
project, use `examples/templates/csharp/mini-compiler`. For reusable parser
library code, use `examples/templates/csharp/library-dsl`.

C# examples prefer source-based parsing with `new Scanner(sourceText)` passed
to generated parser APIs such as `Parser.ParseWithReducerFromLexemeSource(...)` or
instance `ParseRecoveringLexemeSource(...)`. `Scanner.Tokenize(...)` and token-list
parse APIs remain useful for debugging and tests. Reusable examples convert
generated values and diagnostics into domain `ParseResult<T>` values at the
facade boundary; generated code stays container-agnostic, while handwritten
semantics/facades are the right place for policies, services, and DI.

When learning from a C# example, read `*.lf` first, then the handwritten
`Semantics` or `Program.cs` reducer wiring, then any `Parsing` facade. Generated
`*.g.cs` files are intentionally separated under `Generated/`; after running
`make generate`, they show the token enum, scanner DFA tables, parser
ACTION/GOTO tables, and typed reducer context records that the handwritten code
uses.

For a reusable compiler-style starter, use
`examples/templates/csharp/layered-compiler`. It keeps `Generated/` isolated,
puts domain records under `Ast/`, maps typed reducer contexts in `Semantics/`,
exposes `IMiniCompilerParser` plus `ParseResult<T>` from `Parsing/`, and keeps
`Program.cs` as a thin demo entrypoint. It also shows how to inject semantic
policies without making generated code depend on a DI container.

For the recommended handwritten C# reducer, parser facade, reusable library,
dependency-injection, and multi-parser shapes, read
[Handwritten Integration Guide](../../doc/handwritten-integration-guide.md).
