# C DRAW Example

This example parses the DRAW language with generated C scanner/parser code,
builds a handwritten AST through reducer callbacks, executes variables,
figures, `draw`, `repdraw`, math calls, style changes, and drawing primitives,
then writes a PNG image.

```sh
make -C examples/c/draw run
make -C examples/c/draw test
```

The renderer intentionally uses a tiny local RGB/PNG helper instead of external
image libraries, so it is easy to study alongside the generated C API.
