#include "report.hpp"

#include <map>
#include <sstream>

namespace lfdraw {

std::string build_report(const std::string& input_path, const std::string& output_path, const RenderResult& result) {
    std::map<std::string, int> counts;
    for (const auto& op : result.operations) {
        ++counts[op];
    }
    std::ostringstream report;
    report << "DRAW C++ render report\n";
    report << "Source: " << input_path << "\n";
    report << "Output: " << output_path << "\n";
    report << "Canvas: " << result.image.width << "x" << result.image.height << "\n";
    report << "Figures: [";
    for (std::size_t i = 0; i < result.figures.size(); ++i) {
        report << (i == 0 ? "" : ", ") << result.figures[i];
    }
    report << "]\n\nOperation summary:\n";
    for (const auto& item : counts) {
        report << "  " << item.first << ": " << item.second << "\n";
    }
    return report.str();
}

} // namespace lfdraw
