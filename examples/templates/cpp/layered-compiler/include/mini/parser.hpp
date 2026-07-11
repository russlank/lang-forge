#pragma once

#include "mini/ast.hpp"
#include "mini/diagnostics.hpp"

#include <string_view>

namespace mini {

/// Reusable parser facade that hides generated scanner/parser details.
///
/// The generated parser consumes lexemes and reducer callbacks. This facade is
/// the stable application boundary: callers provide source text and receive a
/// domain AST result. Generated table names, reducer-map details, and
/// std::any-compatible parser stack values stay inside the implementation file.
class Parser {
public:
    /// Parses source through a generated scanner lexeme source.
    ///
    /// The returned result owns the AST on success and contains formatted
    /// diagnostics on scanner, syntax, or reducer failure.
    Result<ast::Program> parse(std::string_view source) const;
};

} // namespace mini
