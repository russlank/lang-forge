# Go Mini-Compiler Template

This template is a compact end-to-end compiler pipeline for a tiny `print`
language. `mini.lf` owns lexical and grammar syntax. The Go code owns AST
construction, stack-code lowering, runtime execution, diagnostics, and tests.

Run it from this directory:

```sh
make run
make test
```

To use a standalone LangForge binary instead of `go run`, pass
`LANG_FORGE=/path/to/lang-forge`.
