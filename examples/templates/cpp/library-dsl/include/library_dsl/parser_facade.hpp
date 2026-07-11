#pragma once

#include "library_dsl/ast.hpp"
#include "generated/parser.hpp"

#include <optional>
#include <string>
#include <string_view>
#include <vector>

namespace library_dsl {

/// Domain-level parse result returned by ParserFacade.
struct ParseResult {
    std::optional<Document> value;
    std::vector<std::string> diagnostics;
    bool accepted = false;

    bool success() const noexcept;
};

/// Stable parser facade for applications using the library DSL.
class ParserFacade {
public:
    ParserFacade();

    /// Parses source through the generated scanner lexeme source.
    ParseResult parse(std::string_view source) const;

    /// Compatibility/debug path for callers that already materialized tokens.
    ParseResult parse_tokens(const std::vector<LangForge::Examples::Templates::LibraryDsl::Generated::Lexeme>& tokens) const;

private:
    LangForge::Examples::Templates::LibraryDsl::Generated::ReducerMap reducers_;
};

} // namespace library_dsl
