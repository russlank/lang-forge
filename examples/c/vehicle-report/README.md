# C Vehicle Report Example

This example parses a structured vehicle report and lowers selected reductions
into a normalized text report. It demonstrates a common compiler front-end
pattern: use generated parsing tables for syntax, then keep domain-specific
data extraction in ordinary C code.

The grammar includes named RHS labels and C semantic type declarations. The
demo defaults to the generated typed wrapper in `parser_typed.h`, which
validates typed contexts before delegating to the boxed reducer. Pass `--boxed`
to run the compatibility reducer path directly.

```sh
make -C examples/c/vehicle-report run
make -C examples/c/vehicle-report test
```
