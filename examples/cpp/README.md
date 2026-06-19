# LangForge C++ Examples

The C++ examples are generated-on-demand projects. Each example keeps the
grammar in `.lf`, handwritten semantics in normal C++ source, and generated
scanner/parser output under an ignored `generated/` directory.

Run the calculator example:

```sh
make -C examples/cpp/calc run
```

Run its assertions:

```sh
make -C examples/cpp/calc test
```

The Makefiles invoke LangForge from source by default:

```make
LANG_FORGE ?= $(GO) run ../../../cmd/lang-forge
```

After installing or building a standalone `lang-forge` binary, override that
variable instead:

```sh
make -C examples/cpp/calc LANG_FORGE=lang-forge run
```

Generated C++ output uses conventional filenames: `tokens.hpp`,
`scanner.hpp`, `scanner.cpp`, `parser.hpp`, and `parser.cpp`. Handwritten code
includes generated headers through paths such as `generated/parser.hpp` so IDEs
can see generated types while the generated directory remains the single source
of generated declarations.
