#include "mini/diagnostics.hpp"

namespace mini {

std::vector<std::string> diagnostic_from_exception(const std::exception& ex) {
    return {ex.what()};
}

} // namespace mini
