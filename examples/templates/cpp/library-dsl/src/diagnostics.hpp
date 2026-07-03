#pragma once

#include "generated/parser.hpp"

#include <string>
#include <vector>

namespace library_dsl {

std::vector<std::string> format_diagnostics(const std::vector<LangForge::Examples::Templates::LibraryDsl::Generated::ParseDiagnostic>& diagnostics);

} // namespace library_dsl
