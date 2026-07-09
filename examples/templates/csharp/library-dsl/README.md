# C# Library DSL Template

This template demonstrates a maintainable C# layout around generated
LangForge code:

- `Ast/` contains domain records;
- `Semantics/` maps generated typed reducer contexts to AST construction;
- `Parsing/` exposes `ILibraryDslParser` and `ParseResult<T>`;
- `Generated/` is produced on demand and ignored by Git;
- `Program.cs` is a thin demo entrypoint.

Run it from this directory:

```sh
make test
make run
```

Applications should depend on `ILibraryDslParser`, not on generated parser
types. The template does not reference a dependency-injection package so it can
build with only the .NET SDK; when moving it into an application, register
`LibraryDslParser` with your usual container as the `ILibraryDslParser`
implementation.

`ReducerFactory` lazily builds the pure reducer map once and the parser facade
reuses it. Scanner and parser state are still fresh for each `Parse` call.
