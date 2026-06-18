# C DataKeeper Example

This example uses LangForge-generated C scanner/parser code to parse a
DataKeeper-style script and emit a mock stack-machine execution report.

The C reducer is intentionally side-effect oriented: it records parameters and
commands while still returning simple semantic values for grammar reductions
that need text values. This is a common pattern for compilers that lower source
syntax into an intermediate instruction stream.

```sh
make -C examples/c/datakeeper run
make -C examples/c/datakeeper test
```
