# LangForge C++ Examples

The C++ examples are generated-on-demand projects. Each example keeps the
grammar in `.lf`, handwritten semantics in normal C++ source, and generated
scanner/parser output under an ignored `generated/` directory.

Run one example:

```sh
make -C examples/cpp/calc run
make -C examples/cpp/datakeeper run
make -C examples/cpp/draw run
make -C examples/cpp/vehicle-report run
```

Run generated-code checks:

```sh
make -C examples/cpp/calc test
make -C examples/cpp/datakeeper test
make -C examples/cpp/draw test
make -C examples/cpp/vehicle-report test
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
`scanner.hpp`, `scanner.cpp`, `parser.hpp`, `parser_typed.hpp`, and
`parser.cpp`. Handwritten code includes generated headers through paths such as
`generated/parser.hpp` so IDEs can see generated types while the generated
directory remains the single source of generated declarations.

The examples default to generated typed reducer adapters and keep `--boxed` as
an explicit compatibility mode. The typed path validates named RHS labels and
`ReducerMap` coverage before delegating to the boxed reducer implementation.
When a spec declares structural semantic values as `std::nullptr_t`, reducers
return `nullptr` rather than `{}` when using typed adapters or direct typed
handlers. Boxed-only reducers can tolerate an empty `std::any` only when that
value is never read, but `nullptr` keeps the result consistent with the
declared no-op type.

C++ examples prefer source-based parsing by passing a generated `Scanner` to
`parse_value(scanner, reducers)` or `parse_recovering(scanner)`. Token vectors
remain available for tests and token inspection. Reusable handwritten code
should hide generated headers behind parser facades, use direct typed reducers
where practical, keep `std::any_cast` at compatibility boundaries, and express
ownership with `std::unique_ptr`, `std::variant`, or a domain result type
instead of exposing generated parser stack values.

When learning from a C++ example, read `*.lf` first, then reducer-map creation
such as `make_direct_typed_reducers`, then the parser facade or demo entrypoint.
Generated `scanner.cpp` and `parser.cpp` are useful after that: they show the
scanner DFA tables, static sorted parser ACTION/GOTO tables, and typed reducer
adapter layer that the handwritten code calls.

The DRAW C++ example writes a PNG image with a tiny local encoder rather than
using an external image library.

The Makefiles include shared fragments from `examples/mk` and default to
shared valid fixtures under `examples/testdata`. For a smaller copyable starter
project, use `examples/templates/cpp/mini-compiler`.

For a modern C++ starter with public headers under `include/`, generated output
isolated under `generated/`, a domain-level parser facade, direct typed reducer
handlers, intentional `std::unique_ptr`/`std::variant` ownership, and CMake
integration, use `examples/templates/cpp/layered-compiler`.

For the recommended handwritten C++ reducer map, parser facade, semantic
policy/interface, reusable library, and multi-parser shapes, read
[Handwritten Integration Guide](../../doc/handwritten-integration-guide.md).
