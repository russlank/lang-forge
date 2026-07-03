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

Go examples prefer `ParseWithReducerFromSource(new Scanner(...), reducers)` or
`ParseRecoveringFromSource(new Scanner(...))` in production paths. Token slices
from `Tokenize` are kept for tests and inspection. Normal reducer failures
return `error`; reusable examples keep generated packages behind parser
facades and do not use `panic` for user-input failures.

For the recommended handwritten Go reducer, parser facade, reusable library,
and multi-parser shapes, read
[Handwritten Integration Guide](../../doc/handwritten-integration-guide.md).
