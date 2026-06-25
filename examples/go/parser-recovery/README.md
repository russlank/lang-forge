# Parser Recovery Demo

This runnable teaching example shows how a generated parser can report more
than one syntax error without losing control of the token stream.

The grammar's recovery alternative is:

```lf
Statement : Ident Assign Number Semi
          | error Semi
          ;
```

`error` is a reserved parser-only symbol. When the normal alternative fails,
the runtime pops parser states until it can shift `error`, then discards input
only until `Semi` allows this synchronization production to continue. Every
recovery either pops a state or consumes a token, so malformed input cannot
leave the parser in a non-progressing loop.

The reporting directives keep diagnostics readable:

```lf
%alias Number "number literal"
%group value Ident Number
%hide-expected Semi
```

Aliases replace internal token names in messages. A group replaces two or
more simultaneously expected members with one concept name. Hidden tokens
remain part of the parser table but are omitted from expected-token reports.

Run:

```sh
make run
```

The checked-in input contains invalid assignments on lines 1 and 3. The demo
prints two diagnostics, records each recovery action, and still reports that
the complete input was accepted. Generated files stay under `generated/` and
are ignored by Git.
