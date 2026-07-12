# C++ DataKeeper Example

This example uses LangForge-generated C++17 scanner/parser code to parse a
DataKeeper-style script and emit a mock stack-machine execution report.

The handwritten reducer records parameters and commands while returning text
semantic values for reductions that need them. It mirrors the C and C#
DataKeeper examples while using the generated C++ `ReducerMap` API.

The `.lf` grammar includes named RHS labels and C++ semantic type declarations.
The demo defaults to generated typed adapters in `parser_typed.hpp`, which
validate typed contexts before delegating to the boxed `ReducerMap`. Pass
`--boxed` to run the boxed reducer path directly.

Structural reductions are declared as `std::nullptr_t`. Their reducers return
`nullptr`, not `{}`, in typed mode because `{}` would create an empty
`std::any` rather than a boxed `std::nullptr_t` value for the typed adapter to
validate. The boxed-only path can tolerate an unused empty value, but `nullptr`
keeps both modes consistent.

```sh
make -C examples/cpp/datakeeper run
make -C examples/cpp/datakeeper test
```
