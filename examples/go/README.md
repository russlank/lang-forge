# Go Examples

The Go examples use `lang-forge generate --target go` and keep generated code
under ignored `generated/` directories. Handwritten adapters use the
`langforge_generated` build tag because they import generated packages that do
not exist in a fresh source checkout.

Run one example:

```sh
make -C examples/go/calc run
make -C examples/go/datakeeper run
make -C examples/go/draw run
make -C examples/go/vehicle-report run
```

The Makefiles include shared fragments from `examples/mk` and default to
shared valid fixtures under `examples/testdata`. For a smaller copyable starter
project, use `examples/templates/go/mini-compiler`. For reusable parser
library code, use `examples/templates/go/library-dsl`.

Go examples prefer `ParseWithReducerFromLexemeSource(calc.NewScanner(source), reducers)`
or `ParseRecoveringFromLexemeSource(recovery.NewScanner(source))` for
in-memory text, and `NewReaderScanner(reader, ...)` when reading from files,
stdin, pipes, or virtual sources. Token slices from `Tokenize` are kept for
tests and inspection. Normal reducer failures return `error`; reusable
examples keep generated packages behind parser facades and do not use `panic`
for user-input failures.

Read the Go examples in this order when learning:

1. `*.lf` for the grammar contract.
2. `semantics/reducer.go` or `parser.go` for action-label handling.
3. `cmd/*/main.go` for the thin demo entrypoint.
4. `generated/` after `make generate` for scanner/parser table output.

For the recommended handwritten Go reducer, parser facade, reusable library,
and multi-parser shapes, read
[Handwritten Integration Guide](../../doc/handwritten-integration-guide.md).
