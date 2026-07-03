cmake_minimum_required(VERSION 3.16)
project(langforge_library_dsl_template LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

set(GENERATED_DIR "${CMAKE_CURRENT_SOURCE_DIR}/src/generated")
set(LANG_FORGE "${CMAKE_CURRENT_SOURCE_DIR}/../../../../dist/lang-forge" CACHE FILEPATH "Path to lang-forge binary")

add_custom_command(
  OUTPUT
    "${GENERATED_DIR}/tokens.hpp"
    "${GENERATED_DIR}/scanner.hpp"
    "${GENERATED_DIR}/scanner.cpp"
    "${GENERATED_DIR}/parser.hpp"
    "${GENERATED_DIR}/parser_typed.hpp"
    "${GENERATED_DIR}/parser.cpp"
  COMMAND "${LANG_FORGE}" generate --spec "${CMAKE_CURRENT_SOURCE_DIR}/grammar.lf" --target cpp --out "${GENERATED_DIR}"
  DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/grammar.lf"
  COMMENT "Generating LangForge C++ scanner/parser"
)

add_executable(library-dsl-demo
  src/main.cpp
  src/ast.cpp
  src/diagnostics.cpp
  src/semantics.cpp
  src/parser_facade.cpp
  "${GENERATED_DIR}/scanner.cpp"
  "${GENERATED_DIR}/parser.cpp"
)
target_include_directories(library-dsl-demo PRIVATE include src)
