# C++ DRAW Example

This example parses the DRAW language with generated C++17 scanner/parser code,
builds a handwritten AST through reducer callbacks, executes variables,
figures, `draw`, `repdraw`, math calls, style changes, and drawing primitives,
then writes a PNG image.

The renderer intentionally uses a tiny local PNG writer instead of external
image libraries, so the example stays focused on generated parsing and
reducer-based semantics.

```sh
make -C examples/cpp/draw run
make -C examples/cpp/draw test
```
