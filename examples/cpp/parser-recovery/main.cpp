#include "generated/parser.hpp"

#include <algorithm>
#include <exception>
#include <fstream>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <vector>

namespace recovery = LangForge::Examples::ParserRecovery::Generated;

/// Reads a whole UTF-8 text file into memory.
static std::string read_text_file(const std::string& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open input file: " + path);
    }
    std::ostringstream buffer;
    buffer << input.rdbuf();
    return buffer.str();
}

/// Writes the deterministic demo report beside the executable artifacts.
static void write_text_file(const std::string& path, std::string_view text) {
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

static std::string read_option(std::vector<std::string>& args, const std::string& name, const std::string& fallback) {
    for (auto it = args.begin(); it != args.end(); ++it) {
        if (*it == name) {
            if (it + 1 == args.end()) {
                throw std::runtime_error("missing value for " + name);
            }
            const std::string value = *(it + 1);
            args.erase(it, it + 2);
            return value;
        }
    }
    return fallback;
}

static bool take_flag(std::vector<std::string>& args, const std::string& name) {
    for (auto it = args.begin(); it != args.end(); ++it) {
        if (*it == name) {
            args.erase(it);
            return true;
        }
    }
    return false;
}

/// Formats aliases/groups from `%alias`, `%group`, and `%hide-expected`.
static std::string expected_display(const std::vector<recovery::ExpectedToken>& expected) {
    if (expected.empty()) {
        return "<none>";
    }
    std::string output;
    for (const auto& token : expected) {
        if (!output.empty()) {
            output += ", ";
        }
        output += token.display;
    }
    return output;
}

static std::string build_report(const recovery::ParseResult& result) {
    std::ostringstream report;
    report << "accepted: " << (result.accepted ? "true" : "false") << "\n";
    for (std::size_t index = 0; index < result.diagnostics.size(); ++index) {
        const auto& diagnostic = result.diagnostics[index];
        report << index + 1 << ". "
               << diagnostic.start_line << ":" << diagnostic.start_column
               << " unexpected " << diagnostic.unexpected_display
               << "; expected " << expected_display(diagnostic.expected)
               << "; recovery=" << diagnostic.recovery.kind
               << " discarded=" << diagnostic.recovery.discarded << "\n";
    }
    return report.str();
}

/// Preferred production-style path:
/// source text -> generated scanner lexeme source -> recovering parser.
static recovery::ParseResult parse_source(std::string_view source) {
    recovery::Scanner scanner(source);
    return recovery::parse_recovering(scanner);
}

static void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

static void assert_fixture(const recovery::ParseResult& result) {
    /*
     * Assertions map directly to the grammar fixture:
     * - line 1 recovers by discarding "y" before the semicolon;
     * - line 3 shifts error and immediately synchronizes at Semi;
     * - both diagnostics expect Number via the "number literal" alias.
     */
    require(result.accepted, "fixture should accept after recovery");
    require(result.diagnostics.size() == 2, "expected two diagnostics");
    require(std::any_of(result.diagnostics.begin(), result.diagnostics.end(), [](const recovery::ParseDiagnostic& diagnostic) {
        return diagnostic.recovery.discarded > 0;
    }), "expected one recovery to discard a token");
    require(std::all_of(result.diagnostics.begin(), result.diagnostics.end(), [](const recovery::ParseDiagnostic& diagnostic) {
        return std::any_of(diagnostic.expected.begin(), diagnostic.expected.end(), [](const recovery::ExpectedToken& token) {
            return token.display == "number literal";
        });
    }), "expected number literal diagnostics");
}

int main(int argc, char** argv) {
    try {
        std::vector<std::string> args(argv + 1, argv + argc);
        const bool assert_mode = take_flag(args, "--assert");
        const std::string log_path = read_option(args, "--log", "dist/parser-recovery-cpp-demo.log");
        const std::string input_path = args.empty() ? "input.recovery" : args.front();
        const std::string source = read_text_file(input_path);
        const recovery::ParseResult result = parse_source(source);
        const std::string report = build_report(result);

        write_text_file(log_path, report);
        std::cout << report;
        if (assert_mode) {
            assert_fixture(result);
        }
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
