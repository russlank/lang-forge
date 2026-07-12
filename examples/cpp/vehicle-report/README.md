# C++ Vehicle Report Example

This example parses a structured vehicle report with generated C++17 scanner
and parser code, then lowers selected reductions into a normalized text report.

It demonstrates the same compiler front-end shape as the Go, C#, and C
variants: generated syntax recognition plus handwritten domain extraction.

The grammar includes named RHS labels and C++ semantic type declarations. The
demo defaults to generated typed adapters in `parser_typed.hpp`, which validate
typed contexts before delegating to the boxed `ReducerMap`. Pass `--boxed` to
run the boxed reducer path directly.

Structural reductions are declared as `std::nullptr_t`. Their reducers return
`nullptr`, not `{}`, in typed mode because `{}` would create an empty
`std::any` rather than a boxed `std::nullptr_t` value for the typed adapter to
validate. The boxed-only path can tolerate an unused empty value, but `nullptr`
keeps both modes consistent.

```sh
make -C examples/cpp/vehicle-report run
make -C examples/cpp/vehicle-report test
```
