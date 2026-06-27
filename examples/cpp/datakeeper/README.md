# C++ DataKeeper Example

This example uses LangForge-generated C++17 scanner/parser code to parse a
DataKeeper-style script and emit a mock stack-machine execution report.

The handwritten reducer records parameters and commands while returning text
semantic values for reductions that need them. It mirrors the C and C#
DataKeeper examples while using the generated C++ `ReducerMap` API.

The `.lf` grammar includes named RHS labels matching the Go example. Generated
C++ typed reducer contexts remain future backend-parity work, so the reducer
keeps checked helper functions at the boxed `std::any` boundary.

```sh
make -C examples/cpp/datakeeper run
make -C examples/cpp/datakeeper test
```
