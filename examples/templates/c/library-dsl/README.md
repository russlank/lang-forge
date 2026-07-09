# C Library DSL Template

This template shows a reusable C project layout where generated LangForge code
is hidden behind handwritten headers:

- `ast.h`/`.c` define the domain model, per-parse allocator, and free function;
- `semantics.h`/`.c` map generated typed reducer contexts to AST nodes;
- `parser_facade.h`/`.c` expose one stable parse API;
- `diagnostics.h`/`.c` format generated parser diagnostics;
- `main.c` is a thin demo that reads input, calls the facade, and writes a report;
- `tests.c` is a small smoke-test binary for success, syntax error, reducer
  error, and cleanup behavior;
- `generated/` is produced by `make generate` and ignored by Git.

Run it from this directory:

```sh
make test
make run
```

## Ownership Model

The parser facade is the boundary an application would normally call:

```c
dsl_parse_result result;
dsl_parse_result_init(&result);

if (dsl_parse_source(source, &result)) {
    /* result.document is caller-owned here. */
}

dsl_parse_result_free(&result);
```

Ownership rules:

- The source text passed to `dsl_parse_source` remains caller-owned and must
  stay alive until the call returns.
- The generated scanner and parser state are stack-local inside
  `parser_facade.c`; there is no global mutable parser state.
- Generated syntax diagnostics are stored in `library_dsl_parse_result` and
  freed inside the facade with `library_dsl_parse_result_free`.
- Reducer handlers allocate AST nodes and copied token text from one per-parse
  `dsl_allocator`.
- The typed reducer function-pointer table is static and immutable. Each parse
  copies it and attaches the current semantic context through the generated
  `user` pointer.
- On syntax or reducer failure, the facade destroys that allocator, so partial
  AST nodes built before the error are released.
- On success, the allocator is transferred to the returned `dsl_document`.
  The caller releases the complete tree with `dsl_parse_result_free`.
- Reducer errors are reported by filling the generated `library_dsl_error`
  message and returning `NULL`; normal user-input failures do not call
  `abort`, use global error state, or intentionally leak semantic values.

This template uses a small allocator because it makes failure cleanup easier to
teach: either the final document owns the whole parse tree, or the facade frees
the whole partial tree before returning an error.

## Grammar-To-Code Mapping

The grammar labels values:

```lf
Entry : Set name=Ident Assign value=Value Semi
          {c: entry.set}
      | Enable name=Ident Semi
          {c: entry.enable}
      ;
```

LangForge generates typed contexts such as
`library_dsl_entry_set_reduction`, where reducer code reads `ctx->name` and
`ctx->value` instead of positional parser-stack values. The handwritten
`semantics.c` file turns those generated fields into `dsl_entry` and
`dsl_value` nodes, while `parser_facade.c` keeps generated API details out of
the rest of the application.
