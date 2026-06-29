# C DataKeeper Example

This example uses LangForge-generated C scanner/parser code to parse a
DataKeeper-style script and emit a mock stack-machine execution report.

The C reducer is intentionally side-effect oriented: it records parameters and
commands while still returning simple semantic values for grammar reductions
that need text values. This is a common pattern for compilers that lower source
syntax into an intermediate instruction stream.

The `.lf` grammar includes named RHS labels and C semantic type declarations.
The demo defaults to the generated typed wrapper in `parser_typed.h`, which
validates typed contexts before delegating to the boxed reducer. Pass `--boxed`
to run the compatibility reducer path directly.

```sh
make -C examples/c/datakeeper run
make -C examples/c/datakeeper test
```
