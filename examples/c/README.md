# C Examples

The C examples use `lang-forge generate --target c` and write conventional
generated files into each example's `generated/` directory:

- `tokens.h`
- `scanner.h` and `scanner.c`
- `parser.h`, `parser_typed.h`, and `parser.c`

The compact examples keep most handwritten reducer and demo code in `main.c`
so the generated C API is easy to inspect in one place. For reusable
application code, prefer `examples/templates/c/library-dsl`, which splits the
same responsibilities into `ast`, `semantics`, `parser_facade`, `diagnostics`,
a thin demo, and tests. The generated scanner/parser API is reentrant and does
not use global mutable parse state; examples and templates keep semantic memory
in per-parse state so application code can run multiple parsers independently.

The handwritten C files include generated headers through explicit relative
paths such as `generated/parser.h`, and shared demo helpers through
`../common/demo.h`. This keeps the generated parser header as the single source
of truth while letting IDEs resolve types such as `calc_lexeme` without reading
Makefile `-I` flags. Generate the example once before relying on IDE
navigation.

GCC is the verified compiler in the current workspace. Any C11-capable compiler
should work through the `CC` override. The DRAW example links the math library
with `LDLIBS=-lm` by default.

The examples default to the generated typed reducer wrapper, which validates
named RHS labels and required handlers before delegating to the boxed C reducer.
Pass `--boxed` to any C demo to run the compatibility reducer path directly.
Production paths wrap a generated scanner in a `<prefix>_lexeme_source` and
call source APIs such as `<prefix>_parse_value_lexeme_source_typed`. Token arrays are
kept for compatibility and token-inspection tests.

When learning from a C example, read the files in this order:

1. `*.lf` for the grammar contract.
2. `main.c` or `semantics.c` for the reducer handlers.
3. `parser_facade.c` in templates for reusable ownership and cleanup shape.
4. generated `scanner.c` and `parser.c` after `make generate` for DFA and
   ACTION/GOTO table output.

C callers own cleanup explicitly. Keep source text alive until parsing
returns, free generated token arrays and recovery results with the generated
free functions, and make handwritten parser facades document who owns reducer
semantic values and final AST/document results. The C `library-dsl` template
is the strongest ownership example: it frees partial reducer allocations on
scanner, syntax, or reducer failure and transfers the per-parse allocator to
the final document on success.

Run one example:

```sh
make -C examples/c/calc run
make -C examples/c/datakeeper run
make -C examples/c/draw run
make -C examples/c/vehicle-report run
```

Run generated-code checks:

```sh
make -C examples/c/calc test
make -C examples/c/datakeeper test
make -C examples/c/draw test
make -C examples/c/vehicle-report test
```

If `cc`, `gcc`, or another C compiler is not available through `CC`, the C
Makefiles print a skip message after validation/generation. Set `CC=clang` or
another compiler name when needed.

The Makefiles include shared fragments from `examples/mk` and default to
shared valid fixtures under `examples/testdata`. For a compact compiler
starter, use `examples/templates/c/mini-compiler`. For the recommended
reusable parser-library shape with explicit ownership and cleanup rules, use
`examples/templates/c/library-dsl`.

For the recommended handwritten C reducer, parser adapter, reusable library,
memory-ownership, and multi-parser shapes, read
[Handwritten Integration Guide](../../doc/handwritten-integration-guide.md).
