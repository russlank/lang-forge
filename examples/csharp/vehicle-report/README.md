# C# Vehicle Report Example

This example generates a C# scanner/parser from [vehicle.lf](vehicle.lf). The
handwritten reducer in [Program.cs](Program.cs) builds a vehicle model and
writes a compact text report.

The grammar includes named RHS labels and C# semantic type declarations.
LangForge uses them to generate typed reducer contexts, so the handwritten
reducer can build the report model from properties such as `ctx.Model`,
`ctx.Features`, `ctx.Date`, and `ctx.Description` instead of positional
`object?` values.

Run:

```sh
make run
```

Test:

```sh
make test
```

Generated files live under `Generated/` and use `.g.cs` filenames. Build
outputs and demo logs are ignored.
