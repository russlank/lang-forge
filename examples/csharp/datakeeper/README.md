# C# DataKeeper Example

This example generates a C# scanner/parser from [datakeeper.lf](datakeeper.lf)
and uses a handwritten reducer in [Program.cs](Program.cs) to lower the script
into a small stack-machine-like instruction log.

The grammar uses named RHS labels and C# semantic type declarations in the same
places as the Go version. LangForge turns those into generated typed reducer
contexts, so [Program.cs](Program.cs) can register adapters such as
`SemanticReducerContexts.TypedRunObjectsJob` and implement handlers with
properties like `ctx.Parent`, `ctx.Name`, and `ctx.JobsTag`.

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
