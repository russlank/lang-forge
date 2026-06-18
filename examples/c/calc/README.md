# C Calculator Example

This project generates a reentrant C scanner and parser from `calc.lf`, then
uses a handwritten reducer in `main.c` to evaluate arithmetic expressions.

Run it from the repository root:

```sh
make -C examples/c/calc run
make -C examples/c/calc test
```

`make generate` writes `tokens.h`, `scanner.h`, `scanner.c`, `parser.h`, and
`parser.c` under `generated/`. The handwritten code stays outside that folder
and can be reused as the shape for applications that keep semantic actions in
ordinary C source files.
