# Go Mini-Compiler Template

This template is a compact end-to-end compiler pipeline for a tiny `print`
language. `mini.lf` owns lexical and grammar syntax. The Go code owns AST
construction, stack-code lowering, runtime execution, diagnostics, and tests.

The grammar declares semantic types for `Program`, `Statement`, `Expr`, and the
statement-list nonterminals. It also labels important RHS values, such as
`left=Expr`, `right=Term`, and `token=Number`. LangForge turns those labels into
generated typed contexts such as `AddReduction` and `NumberReduction`, so the
handwritten reducer can read `ctx.Left`, `ctx.Right`, and `ctx.Token` instead of
counting parser stack positions.

Go target types live in the small `model` package because generated code should
not import the command's `main` package. The command wires the generated parser,
typed reducer map, compiler, runtime, and report output together.

Reducer errors are ordinary returned errors. For example, an oversized integer
literal reaches the `number` reducer as a valid token, then returns a semantic
error through `ParseWithReducerFromSource`; the template does not panic for
user-facing reducer failures.

Run it from this directory:

```sh
make run
make test
```

To use a standalone LangForge binary instead of `go run`, pass
`LANG_FORGE=/path/to/lang-forge`.
