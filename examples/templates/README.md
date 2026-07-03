# LangForge Example Templates

Templates are intentionally smaller than the main demos. They show the
copyable shape of a LangForge project:

1. write a `.lf` grammar;
2. generate scanner/parser code on demand;
3. keep handwritten AST, reducer, compiler, runtime, and diagnostics code in
   ordinary source files;
4. test with source inputs rather than checked-in generated artifacts.

The `mini-compiler` template exists for Go, C#, C, and C++. Each target accepts
the same tiny language:

```text
print 1 + 2;
print 40 + 2;
```

The generated parser recognizes the syntax, the reducer builds an AST, the
compiler lowers it to stack instructions, and the mock runtime prints the
results.

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
