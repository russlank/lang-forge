# C Calculator Example

This project generates a reentrant C scanner and parser from `calc.lf`, then
uses a handwritten reducer in `main.c` to evaluate arithmetic expressions.
The grammar matches the Go, C#, and C++ calculator specs, including decimal
numbers such as `4.5`.

Action labels such as `{c: add}` become generated `CALC_ACTION_*` enum values.
`main.c` maps those values to handwritten reducer code and keeps the `void *`
semantic value conversions behind small typed helper functions.

Run it from the repository root:

```sh
make -C examples/c/calc run
make -C examples/c/calc test
```

`make generate` writes `tokens.h`, `scanner.h`, `scanner.c`, `parser.h`, and
`parser.c` under `generated/`. The handwritten code stays outside that folder
and can be reused as the shape for applications that keep semantic actions in
ordinary C source files.
