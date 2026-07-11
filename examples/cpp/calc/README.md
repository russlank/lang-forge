# C++ Calculator Example

This example uses LangForge to generate a C++17 scanner and parser for a small
calculator language. The grammar lives in `calc.lf`; handwritten behavior lives
in `main.cpp`.

Run it:

```sh
make -C examples/cpp/calc run
```

Run assertions:

```sh
make -C examples/cpp/calc test
```

The example is built as C++17. The repository's shared VS Code settings mirror
the Makefile flags for IntelliSense, including the `generated/` include path.
If `std::string_view`, `std::any_cast`, or reducer conversions are underlined in
the editor, regenerate once with `make -C examples/cpp/calc generate` and make
sure the workspace is using the recommended C/C++ extension.

The generated files are written to `generated/`:

- `tokens.hpp`
- `scanner.hpp`
- `scanner.cpp`
- `parser.hpp`
- `parser_typed.hpp`
- `parser.cpp`

The parser rule actions in `calc.lf` are labels, not hard-coded behavior. For
example, `{cpp: add}` becomes `SemanticAction::Add`; `main.cpp` maps that enum
value to the C++ lambda that adds two semantic values. This keeps generated code
separate from handwritten application code while still giving the reducer a
fast, strongly typed dispatch key.

`main.cpp` demonstrates the recommended direct typed path by default:
`parser_typed.hpp` builds contexts such as `AddReduction`, reducer lambdas read
fields such as `ctx.left` and `ctx.right`, and handlers return native `double`
values. The generated adapter boxes results only at the parser boundary. Pass
`--boxed-typed` to exercise the migration adapter that validates typed contexts
before delegating to boxed reducers, or `--boxed` to exercise the older boxed
debug path directly.

The demo evaluates files through `InputStreamScanner` over `std::istream`. The
parser pulls lexemes lazily, and the stream scanner owns copied lexeme text
while parsing so string views remain valid even though the input buffer moves.
Keep the `InputStreamScanner` alive until parsing and reducer code finish. The
in-memory `Scanner` and `tokenize(source)` APIs remain useful when tests or
debugging tools need a token vector before parsing.

The calculator grammar intentionally stays in parity with the Go, C#, and C
versions. The shared sample expression contains a decimal literal and evaluates
to `-2`, so scanner, parser, reducer, and numeric conversion behavior are
exercised together.
