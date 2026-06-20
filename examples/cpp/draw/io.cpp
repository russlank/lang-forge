#include "io.hpp"

#include <filesystem>
#include <fstream>
#include <sstream>
#include <stdexcept>

namespace lfdraw {

std::string read_text_file(const std::string& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open input file: " + path);
    }
    std::ostringstream buffer;
    buffer << input.rdbuf();
    return buffer.str();
}

void ensure_parent_dir(const std::string& path) {
    const std::filesystem::path parent = std::filesystem::path(path).parent_path();
    if (parent.empty()) {
        return;
    }
    std::error_code ec;
    std::filesystem::create_directories(parent, ec);
    if (ec) {
        throw std::runtime_error("cannot create output directory: " + parent.string());
    }
}

void write_text_file(const std::string& path, std::string_view text) {
    ensure_parent_dir(path);
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

} // namespace lfdraw
