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
- `parser.cpp`

The parser rule actions in `calc.lf` are labels, not hard-coded behavior. For
example, `{cpp: add}` becomes `SemanticAction::Add`; `main.cpp` maps that enum
value to the C++ lambda that adds two semantic values. This keeps generated code
separate from handwritten application code while still giving the reducer a
fast, strongly typed dispatch key.
