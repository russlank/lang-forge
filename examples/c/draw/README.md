# C DRAW Example

This example parses the DRAW language with generated C scanner/parser code,
builds a handwritten AST through reducer callbacks, executes variables,
figures, `draw`, `repdraw`, math calls, style changes, and drawing primitives,
then writes a PNG image.

The handwritten C is split by responsibility:

| File | Responsibility |
|---|---|
| `main.c` | CLI options, assertions, orchestration |
| `ast.*` | typed DRAW AST/model and parse allocation context |
| `parser_adapter.*` | generated scanner/parser calls and reducer callbacks |
| `renderer.*` | AST interpretation and raster drawing |
| `report.*` | deterministic console/log report |
| `../common/demo.*` | shared file, arena, text, image, and PNG helpers |

```sh
make -C examples/c/draw run
make -C examples/c/draw test
```

The renderer intentionally uses a tiny local RGB/PNG helper instead of external
image libraries, so it is easy to study alongside the generated C API.

The grammar uses the same named RHS labels and action names as the Go, C#, and
C++ variants. Its C pointer/value types are recorded in
`generated/langforge.actions.json`; shared tail types live in `ast.h` so the
metadata names real application types. Generated C typed contexts in
`generated/parser_typed.h` validate named RHS labels and required handlers
before parsing while the `--boxed` flag keeps the lower-level boxed path
available for comparison.
