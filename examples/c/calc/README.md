# C Calculator Example

This project generates a reentrant C scanner and parser from `calc.lf`, then
uses a handwritten reducer in `main.c` to evaluate arithmetic expressions.
The grammar matches the Go, C#, and C++ calculator specs, including decimal
numbers such as `4.5`.

Action labels such as `{c: add}` become generated `CALC_ACTION_*` enum values.
`main.c` now shows three reducer styles:

- the default direct typed reducer path, where `parser_typed.h` provides
  contexts such as `calc_add_reduction` with fields like `left` and `right`;
- `--boxed-typed`, a migration path where typed contexts validate before
  delegating to the boxed reducer;
- `--boxed`, the boxed/debug path that reads boxed `ctx->values`
  directly.

The direct typed handlers still return `calc_value` pointers because C keeps
semantic ownership explicit. This example stores returned numbers in the demo
arena and releases them with `demo_arena_free`.

The default evaluation path uses `calc_stream_scanner`, not the token-array
string-only `calc_scanner`. `main.c` includes a small `calc_string_stream_read`
callback so the same parser code can be copied to file, stdin, pipe,
editor-buffer, or virtual-file inputs. The generated stream scanner owns copied
visible-lexeme text while parsing; keep it alive until parse/reducer code has
finished and always call `calc_stream_scanner_free`.

Token collections are still useful for inspection and tests:

```c
calc_tokenize(source, &tokens, &count, &error);
calc_parse_tokens(tokens, count, &error);
```

Use the collection path for tests, reports, or token debugging. Prefer
`calc_stream_scanner_lexeme_source_next` with `calc_parse_value_lexeme_source_typed` in
production facades that should pull input on demand.

Run it from the repository root:

```sh
make -C examples/c/calc run
make -C examples/c/calc test
```

`make generate` writes `tokens.h`, `scanner.h`, `scanner.c`, `parser.h`,
`parser_typed.h`, and `parser.c` under `generated/`. The handwritten code stays
outside that folder and can be reused as the shape for applications that keep
semantic actions in ordinary C source files.
