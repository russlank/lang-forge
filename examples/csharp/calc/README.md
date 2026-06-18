# C# Calculator Example

This example generates a C# scanner/parser from [calc.lf](calc.lf), then uses a
handwritten reducer in [Program.cs](Program.cs) to evaluate arithmetic
expressions.

Run:

```sh
make run
```

Test:

```sh
make test
```

The Makefile runs LangForge from source by default:

```sh
/usr/local/go/bin/go run ../../../cmd/lang-forge
```

After `make build` at the repository root, the example can use the standalone
binary instead:

```sh
make LANG_FORGE=../../../dist/lang-forge run
```

Generated files live under `Generated/` and use the conventional `.g.cs`
suffix, such as `Scanner.g.cs`, to make generated C# easy to identify. .NET
build output lives under `bin/` and `obj/`. All of those paths are ignored.
