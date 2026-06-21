# Shared Example Testdata

These fixtures are shared by the runnable examples and the root example gates.
The language-specific projects still keep small local samples for discoverability,
but their Makefiles default to these files so all targets exercise the same
source inputs.

Each family uses the same layout:

- `valid/basic.*` is the normal source used by `make run` and `make test`.
- `invalid/scanner.*` should fail during lexical analysis.
- `invalid/parser.*` should scan successfully and fail during parsing or
  semantic compilation.
- `golden/report.contains` lists stable report fragments checked by
  `make examples-testdata`.

DRAW also has `golden/png.signature`, the expected PNG magic number in hex.
