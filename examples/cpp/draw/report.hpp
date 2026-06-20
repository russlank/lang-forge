#pragma once

#include "renderer.hpp"

#include <string>

namespace lfdraw {

/// Builds the deterministic console/log report for a render.
std::string build_report(const std::string& input_path, const std::string& output_path, const RenderResult& result);

} // namespace lfdraw
