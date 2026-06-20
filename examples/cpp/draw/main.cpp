#include "generated/parser.hpp"
#include "io.hpp"
#include "parser_adapter.hpp"
#include "png_writer.hpp"
#include "renderer.hpp"
#include "report.hpp"

#include <algorithm>
#include <exception>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

namespace draw = LangForge::Examples::Draw::Generated;

namespace {

lfdraw::RenderResult render_source(const std::string& source, const std::string& output_path) {
    lfdraw::Renderer renderer;
    lfdraw::RenderResult result = renderer.render(lfdraw::parse_program(source));
    lfdraw::write_png(output_path, result.image);
    return result;
}

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

int count_operation(const lfdraw::RenderResult& result, const std::string& name) {
    return static_cast<int>(std::count(result.operations.begin(), result.operations.end(), name));
}

void run_assertions(const std::string& source, const std::string& output_path) {
    const lfdraw::RenderResult result = render_source(source, output_path);
    require(result.image.width == 960 && result.image.height == 640, "expected 960x640 canvas");
    require(lfdraw::has_png_signature(output_path), "expected PNG output");
    require(count_operation(result, "line 4 args") == 90, "expected 90 rendered lines");
    require(count_operation(result, "circle 3 args") == 196, "expected 196 rendered circles");
    require(count_operation(result, "box 4 args") == 2, "expected 2 rendered boxes");

    draw::Parser parser(lfdraw::make_reducers());
    const auto tokens = draw::tokenize(source);
    parser.parse_value(tokens);

    try {
        draw::tokenize("canvas 1, @");
        throw std::runtime_error("expected scanner failure");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("no lexical rule") != std::string::npos, "wrong scanner error");
    }

    try {
        draw::parse(draw::tokenize("draw ;"));
        throw std::runtime_error("expected parser failure");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("parse error") != std::string::npos, "wrong parser error");
    }
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
        const std::string output_path = read_option(args, "--output", "dist/sample-cpp.png");
        const std::string log_path = read_option(args, "--log", "dist/draw-cpp-demo.log");
        const std::string input_path = args.empty() ? "sample.draw" : args.front();
        const std::string source = lfdraw::read_text_file(input_path);

        if (assert_mode) {
            run_assertions(source, output_path);
        }

        const lfdraw::RenderResult result = render_source(source, output_path);
        const std::string report = lfdraw::build_report(input_path, output_path, result);
        lfdraw::write_text_file(log_path, report);
        std::cout << report;
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
