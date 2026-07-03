cmake_minimum_required(VERSION 3.16)
project(langforge_layered_compiler_template LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

set(GENERATED_DIR "${CMAKE_CURRENT_SOURCE_DIR}/generated")
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

add_executable(layered-compiler-demo
  src/main.cpp
  src/parser.cpp
  src/compiler.cpp
  src/diagnostics.cpp
  "${GENERATED_DIR}/scanner.cpp"
  "${GENERATED_DIR}/parser.cpp"
)

target_include_directories(layered-compiler-demo PRIVATE
  "${CMAKE_CURRENT_SOURCE_DIR}/include"
  "${CMAKE_CURRENT_SOURCE_DIR}"
)

enable_testing()
add_test(
  NAME layered-compiler-demo-smoke
  COMMAND layered-compiler-demo --assert "${CMAKE_CURRENT_SOURCE_DIR}/input.mini" --log "${CMAKE_CURRENT_BINARY_DIR}/layered-cpp.log"
)
