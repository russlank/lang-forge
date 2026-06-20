#pragma once

#include <string>
#include <string_view>

namespace lfdraw {

/// Reads a UTF-8 text file into memory.
std::string read_text_file(const std::string& path);

/// Creates the parent directory for an output path when one is present.
void ensure_parent_dir(const std::string& path);

/// Writes a complete text file, creating parent directories first.
void write_text_file(const std::string& path, std::string_view text);

} // namespace lfdraw
