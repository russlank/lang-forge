# C++ DRAW Example

This example parses the DRAW language with generated C++17 scanner/parser code,
builds a handwritten AST through reducer callbacks, executes variables,
figures, `draw`, `repdraw`, math calls, style changes, and drawing primitives,
then writes a PNG image.

The renderer intentionally uses a tiny local PNG writer instead of external
image libraries, so the example stays focused on generated parsing and
reducer-based semantics.

The handwritten C++ is split by responsibility:

| File | Responsibility |
|---|---|
| `main.cpp` | CLI options, assertions, orchestration |
| `ast.hpp` | typed DRAW AST/model |
| `parser_adapter.*` | generated scanner/parser calls and reducer map |
| `renderer.*` | AST interpretation and raster drawing |
| `png_writer.*` | dependency-free PNG output |
| `report.*` | deterministic console/log report |
| `io.*` | small file and directory helpers |

```sh
make -C examples/cpp/draw run
make -C examples/cpp/draw test
```

The grammar uses the same named RHS labels and action names as the Go, C#, and
C variants. C++ semantic types such as `lfdraw::ExprPtr` and
`lfdraw::StatementList` are recorded in `generated/langforge.actions.json`.
Generated typed adapters in `generated/parser_typed.hpp` validate reducer-map
coverage and feed handwritten semantic functions through named contexts. Pass
`--boxed` to run the lower-level boxed reducer map directly.
