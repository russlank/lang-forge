# C Examples

The C examples use `lang-forge generate --target c` and write conventional
generated files into each example's `generated/` directory:

- `tokens.h`
- `scanner.h` and `scanner.c`
- `parser.h`, `parser_typed.h`, and `parser.c`

Each example keeps handwritten reducer and demo code in `main.c`. The generated
scanner/parser API is reentrant and does not use global mutable parse state; the
examples keep semantic memory in per-parse arena structs so application code can
run multiple parsers independently.

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
shared valid fixtures under `examples/testdata`. For a smaller copyable starter
project, use `examples/templates/c/mini-compiler`.

For the recommended handwritten C reducer, parser adapter, reusable library,
memory-ownership, and multi-parser shapes, read
[Handwritten Integration Guide](../../doc/handwritten-integration-guide.md).
