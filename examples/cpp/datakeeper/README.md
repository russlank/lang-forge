# C++ DataKeeper Example

This example uses LangForge-generated C++17 scanner/parser code to parse a
DataKeeper-style script and emit a mock stack-machine execution report.

The handwritten reducer records parameters and commands while returning text
semantic values for reductions that need them. It mirrors the C and C#
DataKeeper examples while using the generated C++ `ReducerMap` API.

```sh
make -C examples/cpp/datakeeper run
make -C examples/cpp/datakeeper test
```
