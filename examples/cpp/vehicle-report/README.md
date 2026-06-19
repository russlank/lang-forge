# C++ Vehicle Report Example

This example parses a structured vehicle report with generated C++17 scanner
and parser code, then lowers selected reductions into a normalized text report.

It demonstrates the same compiler front-end shape as the Go, C#, and C
variants: generated syntax recognition plus handwritten domain extraction.

```sh
make -C examples/cpp/vehicle-report run
make -C examples/cpp/vehicle-report test
```
