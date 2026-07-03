#include "diagnostics.hpp"

#include <sstream>

namespace lfgen = LangForge::Examples::Templates::LibraryDsl::Generated;

namespace library_dsl {

std::vector<std::string> format_diagnostics(const std::vector<lfgen::ParseDiagnostic>& diagnostics) {
    std::vector<std::string> out;
    out.reserve(diagnostics.size());
    for (const auto& diagnostic : diagnostics) {
        std::ostringstream line;
        line << diagnostic.start_line << ":" << diagnostic.start_column
             << ": unexpected " << diagnostic.unexpected_display << "; expected ";
        if (diagnostic.expected.empty()) {
            line << "no known continuation";
        } else {
            for (std::size_t i = 0; i < diagnostic.expected.size(); ++i) {
                if (i != 0) {
                    line << ", ";
                }
                line << diagnostic.expected[i].display;
            }
        }
        out.push_back(line.str());
    }
    return out;
}

} // namespace library_dsl
