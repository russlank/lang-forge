#include "generated/parser.hpp"
#include "generated/parser_typed.hpp"

#include <any>
#include <fstream>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <vector>

namespace vehicle = LangForge::Examples::VehicleReport::Generated;

struct Demo {
    std::ostringstream report;
    int features = 0;
    int repairs = 0;
    bool saw_model = false;
    bool saw_license = false;
    bool saw_distance = false;
};

static std::string read_text_file(const std::string& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open input file: " + path);
    }
    std::ostringstream buffer;
    buffer << input.rdbuf();
    return buffer.str();
}

static void write_text_file(const std::string& path, std::string_view text) {
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

static vehicle::Lexeme lexeme_arg(const vehicle::Reduction& ctx, std::size_t index, std::string_view name) {
    // Boxed reducers receive std::any values. This helper keeps the cast and
    // bounds check in one place while typed mode validates labels before
    // delegating to the same semantic implementation.
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing lexeme argument " + std::string(name));
    }
    return std::any_cast<vehicle::Lexeme>(ctx.values.at(index));
}

static std::string text(vehicle::Lexeme lexeme) {
    return std::string(lexeme.text);
}

static std::string unquote(vehicle::Lexeme lexeme) {
    const std::string raw(lexeme.text);
    if (raw.size() >= 2 && raw.front() == '"' && raw.back() == '"') {
        return raw.substr(1, raw.size() - 2);
    }
    return raw;
}

static vehicle::ReducerMap make_reducers(Demo& demo) {
    // The map keys are generated from {cpp: ...} labels in vehicle.lf. Values
    // are ordinary lambdas that build the example's report model.
    //
    // Structural rules in vehicle.lf are declared as std::nullptr_t. Return
    // nullptr for those actions, not {}: vehicle::Value{} is an empty std::any,
    // while vehicle::Value{nullptr} contains the declared std::nullptr_t value
    // that typed_reducer_map_from_boxed validates with std::any_cast.
    auto noop = [](const vehicle::Reduction&) -> vehicle::Value { return nullptr; };
    return vehicle::ReducerMap{
        {vehicle::SemanticAction::Vehicle, noop},
        {vehicle::SemanticAction::Info, noop},
        {vehicle::SemanticAction::FieldModel, [&demo](const vehicle::Reduction& ctx) -> vehicle::Value {
            demo.saw_model = true;
            demo.report << "model: " << unquote(lexeme_arg(ctx, 2, "model literal")) << "\n";
            return nullptr;
        }},
        {vehicle::SemanticAction::FieldLicense, [&demo](const vehicle::Reduction& ctx) -> vehicle::Value {
            demo.saw_license = true;
            demo.report << "license: " << unquote(lexeme_arg(ctx, 2, "license literal")) << "\n";
            return nullptr;
        }},
        {vehicle::SemanticAction::FieldDistance, [&demo](const vehicle::Reduction& ctx) -> vehicle::Value {
            demo.saw_distance = true;
            demo.report << "distance: " << text(lexeme_arg(ctx, 2, "distance literal")) << "\n";
            return nullptr;
        }},
        {vehicle::SemanticAction::FieldFeatures, noop},
        {vehicle::SemanticAction::FeatureItems, noop},
        {vehicle::SemanticAction::FeatureEmpty, noop},
        {vehicle::SemanticAction::FeatureTailMore, noop},
        {vehicle::SemanticAction::FeatureTailEmpty, noop},
        {vehicle::SemanticAction::Feature, [&demo](const vehicle::Reduction& ctx) -> vehicle::Value {
            if (demo.features == 0) {
                demo.report << "features:\n";
            }
            ++demo.features;
            demo.report << "  - " << text(lexeme_arg(ctx, 0, "feature name")) << " = " << unquote(lexeme_arg(ctx, 2, "feature value")) << "\n";
            return nullptr;
        }},
        {vehicle::SemanticAction::FieldRepairs, noop},
        {vehicle::SemanticAction::RepairItems, noop},
        {vehicle::SemanticAction::RepairEmpty, noop},
        {vehicle::SemanticAction::RepairTailMore, noop},
        {vehicle::SemanticAction::RepairTailEmpty, noop},
        {vehicle::SemanticAction::Repair, [&demo](const vehicle::Reduction& ctx) -> vehicle::Value {
            if (demo.repairs == 0) {
                demo.report << "repairs:\n";
            }
            ++demo.repairs;
            demo.report << "  - " << unquote(lexeme_arg(ctx, 3, "repair date")) << ": " << unquote(lexeme_arg(ctx, 7, "repair description")) << "\n";
            return nullptr;
        }},
    };
}

static vehicle::ReducerMap make_typed_reducers(Demo& demo) {
    return vehicle::typed_reducer_map_from_boxed(make_reducers(demo));
}

static std::string parse_source(const std::string& source, Demo& demo, bool typed = true) {
    demo.report << "Vehicle report C++ generated-parser demo\n";
    vehicle::Scanner scanner(source);
    vehicle::parse_value(scanner, typed ? make_typed_reducers(demo) : make_reducers(demo));
    demo.report << "summary: " << demo.features << " features, " << demo.repairs << " repairs\n";
    return demo.report.str();
}

static void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

static void run_assertions(const std::string& source) {
    Demo demo;
    parse_source(source, demo);
    require(demo.saw_model && demo.saw_license && demo.saw_distance && demo.features == 4 && demo.repairs == 3, "unexpected vehicle summary");
    Demo boxed_demo;
    parse_source(source, boxed_demo, false);
    require(boxed_demo.saw_model && boxed_demo.saw_license && boxed_demo.saw_distance && boxed_demo.features == 4 && boxed_demo.repairs == 3, "unexpected boxed vehicle summary");

    try {
        vehicle::tokenize("car = @");
        throw std::runtime_error("expected scanner failure");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("no lexical rule") != std::string::npos, "wrong scanner error");
    }

    try {
        vehicle::parse(vehicle::tokenize("car = {}"));
        throw std::runtime_error("expected parser failure");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("parse error") != std::string::npos, "wrong parser error");
    }
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
        const bool boxed_mode = take_flag(args, "--boxed");
        const std::string log_path = read_option(args, "--log", "dist/vehicle-report-cpp-demo.log");
        const std::string input_path = args.empty() ? "sample.vehicle" : args.front();
        const std::string source = read_text_file(input_path);
        if (assert_mode) {
            run_assertions(source);
        }
        Demo demo;
        const std::string report = parse_source(source, demo, !boxed_mode);
        write_text_file(log_path, report);
        std::cout << report;
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
