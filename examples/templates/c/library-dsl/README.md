# C Library DSL Template

This template shows a C project layout where generated LangForge code is hidden
behind handwritten headers:

- `ast.h`/`.c` define the domain model and ownership/free functions;
- `semantics.h`/`.c` map generated typed reducer contexts to AST nodes;
- `parser_facade.h`/`.c` expose one stable parse API;
- `diagnostics.h`/`.c` format generated parser diagnostics;
- `generated/` is produced by `make generate` and ignored by Git.

Run it from this directory:

```sh
make test
make run
```

The caller owns a successful `dsl_parse_result` and must release it with
`dsl_parse_result_free`.
