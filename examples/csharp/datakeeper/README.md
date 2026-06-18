# C# DataKeeper Example

This example generates a C# scanner/parser from [datakeeper.lf](datakeeper.lf)
and uses a handwritten reducer in [Program.cs](Program.cs) to lower the script
into a small stack-machine-like instruction log.

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
