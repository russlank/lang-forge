# LangForge Example Templates

Templates are intentionally smaller than the main demos. They show the
copyable shape of a LangForge project:

1. write a `.lf` grammar;
2. generate scanner/parser code on demand;
3. keep handwritten AST, reducer, compiler, runtime, and diagnostics code in
   ordinary source files;
4. test with source inputs rather than checked-in generated artifacts.

Two template families exist for Go, C#, C, and C++:

- `mini-compiler` is a compact command-line compiler pipeline;
- `library-dsl` is the recommended real-application starting point, with
  generated parser details hidden behind a stable domain API.

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
