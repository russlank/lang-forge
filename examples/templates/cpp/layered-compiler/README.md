# C++ Layered Compiler Template

This template shows a modern C++17 application layout around generated
LangForge scanner/parser code.

It is intentionally more layered than `../mini-compiler`:

- public application headers live under `include/mini`;
- handwritten implementation lives under `src`;
- generated LangForge output lives under ignored `generated/`;
- `mini::Parser` hides generated scanner/parser details;
- the parser facade returns `mini::Result<mini::ast::Program>`;
- reducers use generated typed contexts directly;
- the compiler walks a domain AST and emits stack-machine instructions;
- `CMakeLists.txt` demonstrates conventional CMake integration.

Run it with the repository Makefile:

```sh
make test
make run
```

Validate the CMake build path:

```sh
make cmake-test
```

`cmake-test` first builds the root `dist/lang-forge` binary, configures a local
build under `dist/cmake-build`, generates parser output through CMake, builds
the demo, and runs the CTest smoke test.

## Ownership Model

The generated C++ parser stores semantic values internally as `std::any`, but
this template keeps `std::any_cast` out of handwritten application code.

The grammar uses copyable semantic handles:

```lf
%semantic cpp type Expr ::mini::ast::ExprId
%semantic cpp type Statement ::mini::ast::Statement
```

Reducers receive generated typed contexts:

```cpp
{lfgen::SemanticAction::Add, lfgen::typed_add(
    [&session](const lfgen::AddReduction& ctx) -> mini::ast::ExprId {
        return session.builder.add(ctx.left, ctx.right);
    })},
```

`mini::ast::ProgramBuilder` owns expression nodes as
`std::unique_ptr<mini::ast::Expr>` during parsing. Reductions pass lightweight
`ExprId` handles on the parser stack. When the final `Program` rule reduces,
the builder transfers the owned nodes into the returned move-only
`mini::ast::Program`.

No `std::shared_ptr` is used because the AST has one owner. If a real compiler
needs shared symbol-table entries or interned strings, introduce sharing at
that explicit boundary rather than for every syntax node.

`std::variant` models closed AST node families:

```cpp
using ExprNode = std::variant<NumberExpr, AddExpr>;
using Statement = std::variant<PrintStatement>;
```

The compiler layer uses visitors over those variants, keeping parser-generated
types out of code generation and runtime execution.

## Generated Boundary

Only `src/parser.cpp` includes generated headers:

```cpp
#include "generated/parser.hpp"
#include "generated/parser_typed.hpp"
```

The public facade in `include/mini/parser.hpp` exposes only domain types:

```cpp
mini::Parser parser;
mini::Result<mini::ast::Program> parsed = parser.parse(source);
```

`Parser::parse` uses the source-based path:

```cpp
lfgen::Scanner scanner(source);
(void)lfgen::parse_value(scanner, make_reducers(session));
```

That means the generated parser pulls lexemes lazily from the generated scanner.
Token vectors remain useful for debugging, but this template teaches the
production path.
