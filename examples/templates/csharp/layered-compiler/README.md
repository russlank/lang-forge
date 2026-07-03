# C# Layered Compiler Template

This template shows a modern C# `net10.0` application layout around generated
LangForge scanner/parser code.

It is intentionally more layered than `../mini-compiler`:

- `Ast/` contains public domain records;
- `Semantics/` maps generated typed reducer contexts to AST construction;
- `Parsing/` exposes `IMiniCompilerParser`, `MiniCompilerParser`, and
  `ParseResult<T>`, with generated diagnostic formatting kept internal;
- `Compilation/` lowers the AST to stack-machine instructions and executes
  the mock runtime;
- `Generated/` is produced on demand and ignored by Git;
- `Program.cs` is a thin demo entrypoint.

Run it from this directory:

```sh
make test
make run
```

## Public Parser Facade

Applications should depend on the interface:

```csharp
IMiniCompilerParser parser = new MiniCompilerParser();
ParseResult<ProgramNode> result = parser.Parse(source);
```

`MiniCompilerParser` uses the preferred source-based generated API internally:

```text
source text
  -> generated Scanner
  -> generated Parser.ParseRecoveringSource
  -> generated typed reducer contexts
  -> Ast.ProgramNode
```

Generated parser types do not appear in the public interface. That keeps the
application boundary stable when generated implementation details change.

## Reducer And DI Pattern

`Semantics/ReducerFactory.cs` is the only handwritten file that maps
`{csharp: ...}` grammar actions to AST-building code. Comments beside each
handler show the associated grammar rule, for example:

```csharp
// Expr : left=Expr Plus right=Term {csharp: add}
[SemanticAction.Add] = TypedAdd(ctx => new AddExprNode(ctx.Left, ctx.Right)),
```

The parser facade accepts `INumberLiteralPolicy`, a domain-level dependency
used by the `number` reducer. A real application that uses
`Microsoft.Extensions.DependencyInjection` can register the facade without this
template taking a package dependency:

```csharp
services.AddSingleton<INumberLiteralPolicy, DefaultNumberLiteralPolicy>();
services.AddSingleton<IMiniCompilerParser, MiniCompilerParser>();
```

The generated recognizer remains container-agnostic. DI is used only at the
handwritten facade/semantic-policy boundary.

## Failure Paths

`make test` runs `Program.cs --assert`, which checks:

- successful parsing, compilation, and execution;
- a syntax error (`print 1 +;`);
- a reducer/semantic error for an oversized integer literal.

Those failures return `ParseResult<T>` diagnostics instead of leaking generated
parser exceptions to callers.
