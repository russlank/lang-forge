#pragma once

#include "ast.hpp"
#include "generated/parser.hpp"

#include <string>

namespace lfdraw {

/// Builds the reducer map that connects generated semantic action IDs to AST construction.
LangForge::Examples::Draw::Generated::ReducerMap make_reducers();

/// Builds typed reducers that validate named RHS labels before AST construction.
LangForge::Examples::Draw::Generated::ReducerMap make_typed_reducers();

/// Scans and parses DRAW source text into a typed AST.
ProgramPtr parse_program(const std::string& source, bool typed = true);

} // namespace lfdraw
