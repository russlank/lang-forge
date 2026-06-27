# C# DataKeeper Example

This example generates a C# scanner/parser from [datakeeper.lf](datakeeper.lf)
and uses a handwritten reducer in [Program.cs](Program.cs) to lower the script
into a small stack-machine-like instruction log.

The grammar uses named RHS labels in the same places as the Go version.
Generated C# typed reducer contexts are still tracked as backend-parity work,
so this example keeps small checked helper functions around the boxed reducer
API.

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
