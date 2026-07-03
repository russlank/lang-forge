# C# Mini-Compiler Template

This template keeps generated C# files under `Generated/` and handwritten
compiler code in `Program.cs`. The reducer builds an AST, the compiler lowers
that AST to stack instructions, and the runtime produces a small execution log.

`mini.lf` declares C# semantic result types with `%semantic csharp type`.
Generated typed reducer adapters expose records such as `AddReduction`,
`PrintReduction`, and `NumberReduction`; `Program.cs` maps those adapters to
ordinary C# functions. This keeps the starter close to real projects without
teaching `ctx.Values[index]` as the main reducer style.

```sh
make run
make test
```
