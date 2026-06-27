# C# Vehicle Report Example

This example generates a C# scanner/parser from [vehicle.lf](vehicle.lf). The
handwritten reducer in [Program.cs](Program.cs) builds a vehicle model and
writes a compact text report.

The grammar includes named RHS labels so generated manifests are useful for
debugging and future typed reducer-context parity. The current C# reducer still
uses checked helper functions over the boxed generated API.

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
