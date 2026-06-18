# C# DRAW Example

This example generates a C# scanner/parser from [draw.lf](draw.lf). The
handwritten reducer and renderer in [Program.cs](Program.cs) build a DRAW AST,
render the script to `dist/sample-csharp.png`, and write a text report.

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
