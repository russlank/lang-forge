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
project, use `examples/templates/csharp/mini-compiler`.
