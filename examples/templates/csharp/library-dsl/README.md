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
types. Register `LibraryDslParser` with your usual dependency-injection
container when moving this template into a service or desktop application.
