# Parser Error Recovery

LangForge generated parsers can continue after selected syntax errors and
return structured diagnostics. Recovery is controlled by the grammar, so a
parser never guesses arbitrary insertion or deletion rules.

## A Minimal Recovery Rule

```lf
%token Ident Number Assign Semi

Statement : Ident Assign Number Semi
          | error Semi
          ;
```

`error` is a reserved parser-only symbol. It is not a scanner token. `Semi` is
the synchronization terminal: after a malformed statement, the parser skips
only as much input as needed to reach a semicolon that lets the recovery
alternative complete.

A recovery alternative must:

- contain `error` exactly once;
- place a terminal after `error`;
- not attach an RHS label to `error`.

These restrictions make the first recovery model explicit and prevent
productions that can repeatedly recover without reaching input.

## Recovery Algorithm

The generated runtimes use the same target-neutral state machine:

```text
on missing action(state, lookahead):
    if not already recovering:
        record diagnostic(state, lookahead, expected[state])
        while stack is not empty:
            if action(top(stack), error) is shift:
                shift synthetic error value
                suppress cascaded diagnostics for 3 real shifts
                resume parsing without consuming lookahead
                break
            pop one state and matching semantic value
        abort if no state can shift error

    otherwise:
        abort if lookahead is end-of-input
        discard one input token
        retry
```

Every search step pops a state, and every token-search step consumes a token.
That progress rule is important: malformed input cannot trap the generated
parser in a recovery loop.

## Readable Expected Tokens

Raw token names are useful to grammar authors but often poor user messages.

```lf
%alias Ident "identifier"
%alias Number "number literal"
%group operator Plus Minus Star Slash
%hide-expected Comma Semi
```

- `%alias` changes the displayed name while retaining the exact terminal.
- `%group` replaces two or more simultaneously expected members with one
  concept such as `operator`. Diagnostics retain the member symbols.
- `%hide-expected` omits punctuation or low-value tokens from reports without
  changing parser behavior.

Expected sets come from the current LR action row. This is deterministic and
useful today. Lookahead-correction-style validation is tracked separately
because it can further improve error timing for merged LALR states.

## Generated APIs

All targets expose:

- an expected-token record;
- a source-rich parse diagnostic;
- a recovery action with discard count;
- a recovery result containing a possibly partial value, diagnostics, and an
  accepted flag.

Go:

```go
result, err := generated.ParseRecovering(tokens)
if err != nil {
    // Internal parser or semantic reducer failure.
}
for _, diagnostic := range result.Diagnostics {
    fmt.Println(diagnostic.UnexpectedDisplay, diagnostic.Expected)
}
```

C#:

```csharp
ParseResult result = Parser.ParseRecovering(tokens);
```

C++:

```cpp
auto result = parse_recovering(tokens);
```

C:

```c
recovery_parse_result result;
recovery_parse_result_init(&result);
if (recovery_parse_recovering(tokens, count, &result, &error)) {
    /* Inspect result.diagnostics and result.accepted. */
}
recovery_parse_result_free(&result);
```

Initialize a C result before its first use, free it after inspection, and free
it before reusing the same variable for another parse. Expected-token metadata
is static generated data; only the diagnostic array is owned by the result.

The established `Parse` and `ParseValue` APIs still treat any syntax
diagnostic as failure. Go returns `*ParseError`, C# throws `ParseException`,
C++ throws `ParseError`, and C returns false with a summary in its error
buffer. Use the recovery API when partial values or multiple diagnostics are
part of the caller's workflow.

## Runnable Example

[The parser recovery demo](../examples/go/parser-recovery) contains two
malformed assignments separated by valid statements. Run:

```sh
make -C examples/go/parser-recovery run
```

It reports both errors, shows their expected token and recovery action, and
still accepts the complete input.
