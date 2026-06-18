# C# Vehicle Report Example

This example generates a C# scanner/parser from [vehicle.lf](vehicle.lf). The
handwritten reducer in [Program.cs](Program.cs) builds a vehicle model and
writes a compact text report.

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
