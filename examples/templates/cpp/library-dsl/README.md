# C++ Library DSL Template

This template demonstrates a reusable C++17 layout:

- public headers live under `include/library_dsl`;
- handwritten source lives under `src`;
- generated LangForge files live under `src/generated`;
- `ParserFacade` hides generated scanner/parser details;
- reducers use generated typed contexts from `parser_typed.hpp`;
- the pure reducer map is a function-local static, so action dispatch wiring is
  created once while scanner/parser state remains per parse.

Run it with the repository Makefile:

```sh
make test
make run
```

The included `CMakeLists.txt` is a conventional starting point for applications
that want to generate LangForge output during a CMake build. Build the root
LangForge binary first or pass `-DLANG_FORGE=/path/to/lang-forge`.
