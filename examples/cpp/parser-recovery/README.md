# C++ Parser Recovery Demo

This example mirrors `examples/go/parser-recovery` with generated C++17 output
under the `LangForge::Examples::ParserRecovery::Generated` namespace. The
grammar uses the reserved parser-only `error` symbol:

```lf
Statement : Ident Assign Number Semi
          | error Semi
          ;
```

The handwritten runner parses from a generated `Scanner` as a `LexemeSource`
and inspects the returned `ParseResult`. Recoverable syntax errors are normal
data, while scanner or internal parser failures remain exceptions.

Run:

```sh
make run
```

The input has two malformed assignments. The report shows accepted status,
expected-token diagnostics, recovery kind, and discarded-token counts.
