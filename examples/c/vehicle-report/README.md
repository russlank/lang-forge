# C Vehicle Report Example

This example parses a structured vehicle report and lowers selected reductions
into a normalized text report. It demonstrates a common compiler front-end
pattern: use generated parsing tables for syntax, then keep domain-specific
data extraction in ordinary C code.

The grammar includes named RHS labels for manifest/debugging clarity. Generated
C typed reducer contexts are still future backend-parity work, so this example
uses small checked helper functions around the boxed reducer values.

```sh
make -C examples/c/vehicle-report run
make -C examples/c/vehicle-report test
```
