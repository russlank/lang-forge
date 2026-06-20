#pragma once

#include "renderer.hpp"

#include <string>

namespace lfdraw {

/// Writes an RGB image as a dependency-free PNG file.
void write_png(const std::string& path, const Image& image);

/// Checks whether a file starts with the PNG signature.
bool has_png_signature(const std::string& path);

} // namespace lfdraw
