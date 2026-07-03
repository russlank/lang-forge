# C# Parser Recovery Demo

This example mirrors `examples/go/parser-recovery` with generated C# output.
The grammar uses the reserved parser-only `error` symbol:

```lf
Statement : Ident Assign Number Semi
          | error Semi
          ;
```

`Semi` is the synchronization token. The demo calls
`Parser.ParseRecoveringFromSource(new Scanner(source))`, inspects the returned
`ParseResult`, and does not rely on exceptions for recoverable syntax errors.

Run:

```sh
make run
```

The input has two malformed assignments. The program prints accepted status,
expected-token diagnostics, recovery kind, and discarded-token counts.
