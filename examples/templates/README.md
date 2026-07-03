# LangForge Example Templates

Templates are intentionally smaller than the main demos. They show the
copyable shape of a LangForge project:

1. write a `.lf` grammar;
2. generate scanner/parser code on demand;
3. keep handwritten AST, reducer, compiler, runtime, and diagnostics code in
   ordinary source files;
4. test with source inputs rather than checked-in generated artifacts.

Two cross-target template families exist for Go, C#, C, and C++:

- `mini-compiler` is a compact command-line compiler pipeline;
- `library-dsl` is the recommended real-application starting point, with
  generated parser details hidden behind a stable domain API.

C# and C++ also have `layered-compiler` starters. The C# version demonstrates
`Ast/`, `Semantics/`, `Parsing/`, `Generated/`, a public
`IMiniCompilerParser`, domain `ParseResult<T>`, and dependency-injection
friendly semantic policy injection. The C++ version is a larger modern C++17
starter that separates public headers from implementation, keeps generated
output isolated, returns a domain-level parser result, demonstrates direct
typed reducer handlers without handwritten `std::any_cast`, and includes both
Makefile and CMake build paths.

The `mini-compiler` template accepts the same tiny language:

```text
print 1 + 2;
print 40 + 2;
```

The generated parser recognizes the syntax, the reducer builds an AST, the
compiler lowers it to stack instructions, and the mock runtime prints the
results.

The `library-dsl` template accepts configuration-like source:

```text
set retries = 3;
set title = "nightly";
enable audit;
```

It splits the code into domain model, semantic reducer, parser facade,
diagnostics, thin demo entrypoint, and tests or smoke assertions. Use it when
you want a copyable shape for a parser embedded in a larger application.
Its facade uses source-based parsing so callers pass source text and receive a
domain result without depending on generated parser stack values.
The C version also demonstrates explicit ownership rules for AST nodes,
generated diagnostics, parse results, reducer errors, and partial semantic
cleanup on failure.

The C++ layered compiler template demonstrates the corresponding C++ ownership
shape: `std::unique_ptr` owns AST expression nodes, `std::variant` models
closed node families, parser reductions exchange lightweight handles, and the
facade returns a move-only domain `Program` wrapped in a C++17 expected-like
`Result`.

The C# layered compiler template demonstrates the corresponding C# application
shape: domain records live under `Ast/`, generated types stay out of public
interfaces, `MiniCompilerParser` accepts injectable semantic policy services,
and `Program.cs` is only a thin demo over the reusable parser facade.

Each template uses the current recommended LangForge reducer style:

- `%semantic <target> type` declarations describe the semantic type produced by
  each parser nonterminal;
- grammar alternatives use named RHS labels such as `left=Expr`,
  `right=Term`, `expr=Expr`, and `token=Number`;
- handwritten reducer code consumes generated typed reducer contexts instead of
  indexing parser stack values manually;
- reducer failures are returned through the generated parse API with useful
  rule/action/field context rather than handled with panic-style helpers;
- `langforge.actions.json` records the typed action contract for review and
  tooling.
