# C Parser Recovery Demo

This example mirrors `examples/go/parser-recovery` with generated C output.
The grammar uses the reserved parser-only `error` symbol:

```lf
Statement : Ident Assign Number Semi
          | error Semi
          ;
```

The handwritten runner initializes a `recovery_parse_result`, parses from a
generated scanner-backed `recovery_lexeme_source`, prints diagnostics, writes
the same report to `dist/parser-recovery-c-demo.log`, and frees the result.

Run:

```sh
make run
```

The input has two malformed assignments. The report shows accepted status,
expected-token diagnostics, recovery kind, and discarded-token counts.
