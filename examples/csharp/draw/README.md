# C# DRAW Example

This example generates a C# scanner/parser from [draw.lf](draw.lf). The
handwritten code builds a DRAW AST, renders the script to
`dist/sample-csharp.png`, and writes a text report.

The example is intentionally split by responsibility so it is easier to copy
into real projects:

| File | Responsibility |
|---|---|
| `Program.cs` | CLI options, file IO, assertion orchestration |
| `ParserAdapter.cs` | generated scanner/parser calls and reducer dispatch |
| `Ast.cs` | typed DRAW AST and color model |
| `DrawRenderer.cs` | AST interpretation and raster drawing |
| `ImageBuffer.cs` | in-memory RGB pixel storage |
| `PngWriter.cs` | dependency-free PNG output |
| `ReportWriter.cs` | deterministic console/log report |

Run:

```sh
make run
```

Test:

```sh
make test
```

Generated files live under `Generated/` and use `.g.cs` filenames. Build
outputs, generated PNG files, and demo logs are ignored.

The grammar uses the same named RHS labels and action names as the Go, C, and
C++ DRAW specs. Target-specific C# semantic types are recorded in
`Generated/langforge.actions.json`. The handwritten adapter uses generated C#
typed reducer contexts and coverage validation, so reductions read from named
properties such as `ctx.Width`, `ctx.Height`, `ctx.Target`, and `ctx.Tail`
instead of counting parser stack positions.
