# C Mini-Compiler Template

This is a single-file C11 starter showing the generated parser boundary and a
small handwritten compiler pipeline. It intentionally avoids a framework:
generated files live in `generated/`, while `main.c` owns AST allocation,
lowering, runtime execution, diagnostics, and file IO.

The grammar declares C semantic result types for the AST pointers. The generated
`parser_typed.h` header provides contexts such as
`mini_compiler_add_reduction`, where named grammar labels become fields like
`left` and `right`. `main.c` calls `mini_compiler_parse_value_lexeme_source_typed` so
new projects start from the typed reducer API.

Reducer failures are reported by filling `mini_compiler_error` and returning
`NULL`, not by aborting. The self-test exercises an oversized integer literal
that reaches the typed `number` handler and returns an action/label-rich error.

```sh
make run
make test
```
