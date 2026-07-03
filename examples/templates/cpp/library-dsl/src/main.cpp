#include "library_dsl/parser_facade.hpp"

#include <fstream>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>

static std::string read_text_file(const std::string& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open input file: " + path);
    }
    std::ostringstream buffer;
    buffer << input.rdbuf();
    return buffer.str();
}

static void write_text_file(const std::string& path, const std::string& text) {
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

static std::string report(const std::string& input_path, const library_dsl::Document& document) {
    std::ostringstream out;
    out << "Library DSL C++ template: " << input_path << "\n";
    for (const auto& entry : document.entries) {
        out << "  " << library_dsl::entry_kind_name(entry.kind) << " " << entry.name << " = " << entry.value.format() << "\n";
    }
    return out.str();
}

static void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

static void run_assertions(const library_dsl::ParserFacade& parser) {
    const auto parsed = parser.parse("set retries = 3;\nset title = \"nightly\";\nenable audit;");
    require(parsed.success(), parsed.diagnostics.empty() ? "parse failed" : parsed.diagnostics.front());
    const auto settings = parsed.value->settings();
    require(settings.at("retries").number == 3, "unexpected retries value");
    require(settings.at("title").text == "nightly", "unexpected title value");
    require(settings.at("audit").boolean, "expected audit flag");
    require(!parser.parse("set retries = ;").success(), "expected parser failure");
    require(!parser.parse("set retries = 999999999999999999999999;").success(), "expected reducer failure");
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

int main(int argc, char** argv) {
    try {
        std::vector<std::string> args(argv + 1, argv + argc);
        const bool assert_mode = take_flag(args, "--assert");
        const std::string log_path = read_option(args, "--log", "dist/library-cpp.log");
        const std::string input_path = args.empty() ? "input.dsl" : args.front();
        const std::string source = read_text_file(input_path);
        library_dsl::ParserFacade parser;
        if (assert_mode) {
            run_assertions(parser);
        }
        const auto parsed = parser.parse(source);
        if (!parsed.success()) {
            throw std::runtime_error(parsed.diagnostics.empty() ? "parse failed" : parsed.diagnostics.front());
        }
        const auto text = report(input_path, *parsed.value);
        std::cout << text;
        write_text_file(log_path, text);
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
