#include "mini/compiler.hpp"
#include "mini/parser.hpp"

#include <fstream>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>

namespace {

std::string read_text_file(const std::string& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open input file: " + path);
    }
    std::ostringstream buffer;
    buffer << input.rdbuf();
    return buffer.str();
}

void write_text_file(const std::string& path, const std::string& text) {
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

mini::ast::Program parse_or_throw(const mini::Parser& parser, std::string_view source) {
    auto parsed = parser.parse(source);
    if (!parsed) {
        const auto& diagnostics = parsed.diagnostics();
        throw std::runtime_error(diagnostics.empty() ? "parse failed" : diagnostics.front());
    }
    return parsed.take_value();
}

void run_assertions(const mini::Parser& parser, std::string_view source) {
    const auto output = mini::compiler::execute(mini::compiler::compile(parse_or_throw(parser, source)));
    require(output.size() == 2 && output[0] == 3 && output[1] == 42, "unexpected template output");

    auto syntax = parser.parse("print 1 +;");
    require(!syntax.ok(), "expected parser failure");
    require(!syntax.diagnostics().empty() && syntax.diagnostics().front().find("parse error") != std::string::npos,
        "wrong parser diagnostic");

    auto reducer = parser.parse("print 999999999999999999999999;");
    require(!reducer.ok(), "expected reducer failure");
    require(!reducer.diagnostics().empty() &&
            reducer.diagnostics().front().find("action number") != std::string::npos &&
            reducer.diagnostics().front().find("label token") != std::string::npos,
        "wrong reducer diagnostic");
}

std::string read_option(std::vector<std::string>& args, const std::string& name, const std::string& fallback) {
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

bool take_flag(std::vector<std::string>& args, const std::string& name) {
    for (auto it = args.begin(); it != args.end(); ++it) {
        if (*it == name) {
            args.erase(it);
            return true;
        }
    }
    return false;
}

} // namespace

int main(int argc, char** argv) {
    try {
        std::vector<std::string> args(argv + 1, argv + argc);
        const bool assert_mode = take_flag(args, "--assert");
        const std::string log_path = read_option(args, "--log", "dist/layered-cpp.log");
        const std::string input_path = args.empty() ? "input.mini" : args.front();
        const std::string source = read_text_file(input_path);
        mini::Parser parser;

        if (assert_mode) {
            run_assertions(parser, source);
        }

        const auto program = parse_or_throw(parser, source);
        const auto code = mini::compiler::compile(program);
        const auto output = mini::compiler::execute(code);
        const auto text = mini::compiler::format_report(input_path, code, output);
        std::cout << text;
        write_text_file(log_path, text);
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
