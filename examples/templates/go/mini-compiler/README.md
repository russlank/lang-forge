# Go Mini-Compiler Template

This template is a compact end-to-end compiler pipeline for a tiny `print`
language. `mini.lf` owns lexical and grammar syntax. The Go code owns AST
construction, stack-code lowering, runtime execution, diagnostics, and tests.

The grammar labels important RHS values, such as `left=Expr`, `right=Term`,
and `token=Number`. The handwritten reducer uses `Reduction.ValueFor` to read
those labels, which is the lightest-weight path before introducing generated
typed contexts.

Run it from this directory:

```sh
make run
make test
```

To use a standalone LangForge binary instead of `go run`, pass
`LANG_FORGE=/path/to/lang-forge`.
