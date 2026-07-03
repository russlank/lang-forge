#include "library_dsl/parser_facade.hpp"

#include "diagnostics.hpp"
#include "semantics.hpp"

#include <any>
#include <stdexcept>

namespace lfgen = LangForge::Examples::Templates::LibraryDsl::Generated;

namespace library_dsl {

bool ParseResult::success() const noexcept {
    return accepted && diagnostics.empty() && value.has_value();
}

ParserFacade::ParserFacade() : reducers_(make_reducers()) {}

ParseResult ParserFacade::parse(std::string_view source) const {
    try {
        lfgen::Scanner scanner(source);
        lfgen::Parser parser(reducers_);
        auto result = parser.parse_recovering(scanner);
        if (!result.accepted || !result.diagnostics.empty()) {
            return ParseResult{std::nullopt, format_diagnostics(result.diagnostics), false};
        }
        return ParseResult{std::any_cast<Document>(result.value), {}, true};
    } catch (const std::exception& ex) {
        return ParseResult{std::nullopt, {ex.what()}, false};
    }
}

ParseResult ParserFacade::parse_tokens(const std::vector<lfgen::Lexeme>& tokens) const {
    try {
        lfgen::Parser parser(reducers_);
        auto result = parser.parse_recovering(tokens);
        if (!result.accepted || !result.diagnostics.empty()) {
            return ParseResult{std::nullopt, format_diagnostics(result.diagnostics), false};
        }
        return ParseResult{std::any_cast<Document>(result.value), {}, true};
    } catch (const std::exception& ex) {
        return ParseResult{std::nullopt, {ex.what()}, false};
    }
}

} // namespace library_dsl
