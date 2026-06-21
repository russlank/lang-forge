# C Mini-Compiler Template

This is a single-file C11 starter showing the generated parser boundary and a
small handwritten compiler pipeline. It intentionally avoids a framework:
generated files live in `generated/`, while `main.c` owns AST allocation,
lowering, runtime execution, diagnostics, and file IO.

```sh
make run
make test
```
