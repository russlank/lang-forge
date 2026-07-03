# C++ Mini-Compiler Template

This C++17 starter uses small AST structs, generated parser reducer hooks, a
small instruction stream, and a mock stack runtime. It is meant to be copied and
renamed when starting a new DSL.

The grammar declares semantic result types such as `::Program`,
`std::vector<::Statement>`, and `::ExprPtr`. The generated `parser_typed.hpp`
adapters expose typed contexts such as `mini::AddReduction` and
`mini::NumberReduction`, so handwritten handlers read `ctx.left`, `ctx.right`,
and `ctx.token` instead of using `std::any` positions directly.

User-facing reducer failures should be thrown as ordinary parser-call errors
with grammar context. The self-test includes an oversized integer literal that
the scanner accepts and the typed `number` reducer rejects with rule/action/label
details.

```sh
make run
make test
```
