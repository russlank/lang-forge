#include "generated/parser.hpp"
#include "generated/parser_typed.hpp"

#include <any>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <vector>

namespace dks = LangForge::Examples::DataKeeper::Generated;

struct Demo {
    std::ostringstream report;
    int parameters = 0;
    int commands = 0;
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

static dks::Lexeme lexeme_arg(const dks::Reduction& ctx, std::size_t index, std::string_view name) {
    // Boxed reducers still receive std::any values. Keep casts in small helpers
    // whose names mirror RHS labels; typed mode validates those labels before
    // delegating to this compatibility implementation.
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing lexeme argument " + std::string(name));
    }
    return std::any_cast<dks::Lexeme>(ctx.values.at(index));
}

static std::string string_arg(const dks::Reduction& ctx, std::size_t index, std::string_view name) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing string argument " + std::string(name));
    }
    return std::any_cast<std::string>(ctx.values.at(index));
}

static std::string text(dks::Lexeme lexeme) {
    return std::string(lexeme.text);
}

static std::string decode_literal(dks::Lexeme lexeme) {
    const std::string raw(lexeme.text);
    if (raw.size() >= 4 && raw.rfind("#{", 0) == 0 && raw.substr(raw.size() - 2) == "#}") {
        std::string out;
        for (std::size_t i = 2; i + 2 < raw.size(); ++i) {
            if (raw[i] == '#' && i + 1 < raw.size() && raw[i + 1] == '#') {
                out.push_back('#');
                ++i;
            } else {
                out.push_back(raw[i]);
            }
        }
        return out;
    }
    if (raw.size() >= 2 && raw.front() == '"' && raw.back() == '"') {
        std::string out;
        for (std::size_t i = 1; i + 1 < raw.size(); ++i) {
            if (raw[i] == '\\' && i + 1 < raw.size() - 1) {
                ++i;
            }
            out.push_back(raw[i]);
        }
        return out;
    }
    return raw;
}

static void append_parameter(Demo& demo, const std::string& name) {
    ++demo.parameters;
    demo.report << "  param " << std::setw(2) << std::left << demo.parameters << " " << name << "\n";
}

static void append_command(Demo& demo, const std::string& kind, const std::string& a, const std::string& b = "", const std::string& c = "") {
    ++demo.commands;
    demo.report << "  " << std::right << std::setw(2) << std::setfill('0') << demo.commands << std::setfill(' ')
                << " " << std::setw(14) << std::left << kind << std::right;
    if (!a.empty()) {
        demo.report << " " << a;
    }
    if (!b.empty()) {
        demo.report << " | " << b;
    }
    if (!c.empty()) {
        demo.report << " | " << c;
    }
    demo.report << "\n";
}

static dks::ReducerMap make_reducers(Demo& demo) {
    // SemanticAction values are generated from {cpp: ...} labels in the
    // grammar. ReducerMap keeps semantic dispatch data-driven and leaves all
    // domain behavior in handwritten C++.
    //
    // Structural rules in datakeeper.lf are declared as std::nullptr_t. Return
    // nullptr for those actions, not {}: dks::Value{} is an empty std::any,
    // while dks::Value{nullptr} contains the declared std::nullptr_t value that
    // typed_reducer_map_from_boxed validates with std::any_cast.
    auto noop = [](const dks::Reduction&) -> dks::Value { return nullptr; };
    auto pass = [](const dks::Reduction& ctx) -> dks::Value {
        return ctx.values.empty() ? dks::Value{} : ctx.values.at(0);
    };
    return dks::ReducerMap{
        {dks::SemanticAction::ProgramWithParameters, noop},
        {dks::SemanticAction::ProgramNoParameters, noop},
        {dks::SemanticAction::ParametersList, noop},
        {dks::SemanticAction::ParametersDecl, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_parameter(demo, text(lexeme_arg(ctx, 0, "parameter name")));
            return nullptr;
        }},
        {dks::SemanticAction::ParametersTailMore, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_parameter(demo, text(lexeme_arg(ctx, 1, "parameter name")));
            return nullptr;
        }},
        {dks::SemanticAction::ParametersTailEmpty, noop},
        {dks::SemanticAction::CommandBlock, noop},
        {dks::SemanticAction::Statements, noop},
        {dks::SemanticAction::StatementsTailMore, noop},
        {dks::SemanticAction::StatementsTailEmpty, noop},
        {dks::SemanticAction::StatementPass, pass},
        {dks::SemanticAction::ValueString, [](const dks::Reduction& ctx) -> dks::Value {
            return decode_literal(lexeme_arg(ctx, 0, "string literal"));
        }},
        {dks::SemanticAction::ValueNumber, [](const dks::Reduction& ctx) -> dks::Value {
            return text(lexeme_arg(ctx, 0, "number literal"));
        }},
        {dks::SemanticAction::ValueIdent, [](const dks::Reduction& ctx) -> dks::Value {
            return "$" + text(lexeme_arg(ctx, 0, "identifier value"));
        }},
        {dks::SemanticAction::Assign, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_command(demo, "assign", text(lexeme_arg(ctx, 0, "assignment name")), string_arg(ctx, 2, "assignment value"));
            return nullptr;
        }},
        {dks::SemanticAction::Replace, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_command(demo, "replace", text(lexeme_arg(ctx, 2, "replace target")), string_arg(ctx, 4, "old value"), string_arg(ctx, 6, "new value"));
            return nullptr;
        }},
        {dks::SemanticAction::Sqlrun, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_command(demo, "sqlrun", string_arg(ctx, 2, "instance"), string_arg(ctx, 4, "script"));
            return nullptr;
        }},
        {dks::SemanticAction::AddObject, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_command(demo, "addobject", string_arg(ctx, 2, "parent"), string_arg(ctx, 4, "xml"));
            return nullptr;
        }},
        {dks::SemanticAction::RemoveObject, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_command(demo, "removeobject", string_arg(ctx, 2, "parent"), string_arg(ctx, 4, "name"));
            return nullptr;
        }},
        {dks::SemanticAction::RunObjectsJob, [&demo](const dks::Reduction& ctx) -> dks::Value {
            append_command(demo, "runobjectsjob", string_arg(ctx, 2, "parent"), string_arg(ctx, 4, "name"), string_arg(ctx, 6, "jobs tag"));
            return nullptr;
        }},
    };
}

static dks::ReducerMap make_typed_reducers(Demo& demo) {
    return dks::typed_reducer_map_from_boxed(make_reducers(demo));
}

static std::string parse_source(const std::string& source, Demo& demo, bool typed = true) {
    demo.report << "DataKeeper C++ mock compiler\nparameters:\n";
    dks::parse_value(dks::tokenize(source), typed ? make_typed_reducers(demo) : make_reducers(demo));
    demo.report << "summary: " << demo.parameters << " parameters, " << demo.commands << " mock stack instructions\n";
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
    require(demo.parameters == 4 && demo.commands == 8, "unexpected DataKeeper summary");
    Demo boxed_demo;
    parse_source(source, boxed_demo, false);
    require(boxed_demo.parameters == 4 && boxed_demo.commands == 8, "unexpected boxed DataKeeper summary");

    try {
        dks::tokenize("begin @ end");
        throw std::runtime_error("expected scanner failure");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("no lexical rule") != std::string::npos, "wrong scanner error");
    }

    try {
        dks::parse(dks::tokenize("begin end"));
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
        const std::string log_path = read_option(args, "--log", "dist/datakeeper-cpp-demo.log");
        const std::string input_path = args.empty() ? "sample.dks" : args.front();
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
