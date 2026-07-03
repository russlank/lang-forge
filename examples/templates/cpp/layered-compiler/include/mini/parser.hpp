#pragma once

#include "mini/ast.hpp"
#include "mini/diagnostics.hpp"

#include <string_view>

namespace mini {

/// Reusable parser facade that hides generated scanner/parser details.
class Parser {
public:
    /// Parses source through a generated scanner token source.
    Result<ast::Program> parse(std::string_view source) const;
};

} // namespace mini
