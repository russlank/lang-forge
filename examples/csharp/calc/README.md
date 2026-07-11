# C# Calculator Example

This example generates a C# scanner/parser from [calc.lf](calc.lf), then uses a
handwritten reducer in [Program.cs](Program.cs) to evaluate arithmetic
expressions.

The grammar intentionally matches the Go, C, and C++ calculator specs: numbers
may contain a fractional part, and the default sample evaluates
`1 + 2 * (3 - 4.5)` to `-2`.

Action labels such as `{csharp: add}` become generated `SemanticAction` enum
values. They do not contain arithmetic by themselves; [Program.cs](Program.cs)
maps those enum values to generated typed reducer adapters such as
`SemanticReducerContexts.TypedAdd`. The handwritten handler receives an
`AddReduction` record with named properties like `Left` and `Right`, so grammar
changes are easier to review than positional `object?` casts.

The demo uses the generated reader/stream scanner APIs for the production path:
`Scanner.FromStream` reads `input.calc` on demand, and
`Parser.ParseWithReducerFromLexemeSource` pulls tokens from that scanner. The
assertion suite also exercises `Scanner.FromTextReader`, one-character read
buffers, reader failures, and buffered-lexeme limits. The collection APIs remain
available for debugging:

```csharp
var tokens = Scanner.Tokenize(sourceText);
var value = Parser.ParseWithReducer(tokens, reducers);
```

Use the collection path when you want to inspect or snapshot tokens. Prefer
`FromTextReader` or `FromStream` inside reusable facades that parse files,
stdin, pipes, editor buffers, or other demand-fed inputs.

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
