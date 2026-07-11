#include "generated/parser.hpp"
#include "generated/parser_typed.hpp"

#include <any>
#include <atomic>
#include <cmath>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

namespace lfcalc = LangForge::Examples::Calc::Generated;

enum class ReducerMode {
    DirectTyped,
    BoxedToTyped,
    Boxed,
};

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

/// Writes a complete text report, creating or replacing the destination file.
static void write_text_file(const std::string& path, std::string_view text) {
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

/// Returns the numeric semantic value at the requested reduction argument.
static double number_arg(const lfcalc::Reduction& ctx, std::size_t index, std::string_view name) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing numeric argument " + std::string(name));
    }
    try {
        return std::any_cast<double>(ctx.values.at(index));
    } catch (const std::bad_any_cast&) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " argument " + std::to_string(index + 1) + " is not numeric");
    }
}

/// Returns the generated scanner lexeme at the requested reduction argument.
static lfcalc::Lexeme lexeme_arg(const lfcalc::Reduction& ctx, std::size_t index, std::string_view name) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing lexeme argument " + std::string(name));
    }
    try {
        return std::any_cast<lfcalc::Lexeme>(ctx.values.at(index));
    } catch (const std::bad_any_cast&) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " argument " + std::to_string(index + 1) + " is not a lexeme");
    }
}

/// Builds the calculator's handwritten semantics.
///
/// The grammar contains labels such as `{cpp: add}`. LangForge turns those
/// labels into `SemanticAction` enum values, and this reducer map connects each
/// generated action ID to ordinary C++ code. Keeping this as a map avoids a long
/// reduction switch and makes it obvious which grammar actions are implemented.
///
/// This boxed reducer map is intentionally kept as migration material. New C++
/// code should prefer `make_direct_typed_reducers`, where generated contexts
/// expose named fields and handwritten handlers return native semantic types.
static const lfcalc::ReducerMap& make_boxed_reducers() {
    static const lfcalc::ReducerMap reducers{
        {lfcalc::SemanticAction::Start, [](const lfcalc::Reduction& ctx) -> lfcalc::Value { return ctx.values.at(0); }},
        {lfcalc::SemanticAction::Pass, [](const lfcalc::Reduction& ctx) -> lfcalc::Value { return ctx.values.at(0); }},
        {lfcalc::SemanticAction::Group, [](const lfcalc::Reduction& ctx) -> lfcalc::Value { return ctx.values.at(1); }},
        {lfcalc::SemanticAction::Number, [](const lfcalc::Reduction& ctx) -> lfcalc::Value {
            const auto lexeme = lexeme_arg(ctx, 0, "number");
            return std::stod(std::string(lexeme.text));
        }},
        {lfcalc::SemanticAction::Negate, [](const lfcalc::Reduction& ctx) -> lfcalc::Value {
            return -number_arg(ctx, 1, "operand");
        }},
        {lfcalc::SemanticAction::Add, [](const lfcalc::Reduction& ctx) -> lfcalc::Value {
            return number_arg(ctx, 0, "left operand") + number_arg(ctx, 2, "right operand");
        }},
        {lfcalc::SemanticAction::Subtract, [](const lfcalc::Reduction& ctx) -> lfcalc::Value {
            return number_arg(ctx, 0, "left operand") - number_arg(ctx, 2, "right operand");
        }},
        {lfcalc::SemanticAction::Multiply, [](const lfcalc::Reduction& ctx) -> lfcalc::Value {
            return number_arg(ctx, 0, "left operand") * number_arg(ctx, 2, "right operand");
        }},
        {lfcalc::SemanticAction::Divide, [](const lfcalc::Reduction& ctx) -> lfcalc::Value {
            const double right = number_arg(ctx, 2, "right operand");
            if (right == 0.0) {
                throw std::runtime_error("division by zero");
            }
            return number_arg(ctx, 0, "left operand") / right;
        }},
    };
    return reducers;
}

/// Builds direct typed reducers, the recommended path for new C++ code.
///
/// Each helper from `parser_typed.hpp` constructs a generated context such as
/// `AddReduction` or `NumberReduction`. Reducer lambdas read named fields
/// (`ctx.left`, `ctx.right`, `ctx.token`) and return the declared native
/// semantic type (`double` here). LangForge boxes the result only at the parser
/// boundary, so the recommended handwritten path does not need std::any_cast.
///
/// Grammar alternatives implemented here:
/// - S : value=Expr {cpp: start}
/// - Expr : left=Expr Plus right=Term {cpp: add}
/// - Expr : left=Expr Minus right=Term {cpp: subtract}
/// - Expr : value=Term {cpp: pass}
/// - Term : left=Term Mul right=Factor {cpp: multiply}
/// - Term : left=Term Div right=Factor {cpp: divide}
/// - Factor : token=Number {cpp: number}
/// - Factor : LParen value=Expr RParen {cpp: group}
/// - Factor : Minus value=Factor {cpp: negate}
static const lfcalc::ReducerMap& make_direct_typed_reducers() {
    static const lfcalc::ReducerMap reducers{
        {lfcalc::SemanticAction::Start, lfcalc::typed_start([](const lfcalc::StartReduction& ctx) -> double {
            return ctx.value;
        })},
        {lfcalc::SemanticAction::Pass, lfcalc::typed_pass([](const lfcalc::PassReduction& ctx) -> double {
            return ctx.value;
        })},
        {lfcalc::SemanticAction::Group, lfcalc::typed_group([](const lfcalc::GroupReduction& ctx) -> double {
            return ctx.value;
        })},
        {lfcalc::SemanticAction::Number, lfcalc::typed_number([](const lfcalc::NumberReduction& ctx) -> double {
            return std::stod(std::string(ctx.token.text));
        })},
        {lfcalc::SemanticAction::Negate, lfcalc::typed_negate([](const lfcalc::NegateReduction& ctx) -> double {
            return -ctx.value;
        })},
        {lfcalc::SemanticAction::Add, lfcalc::typed_add([](const lfcalc::AddReduction& ctx) -> double {
            return ctx.left + ctx.right;
        })},
        {lfcalc::SemanticAction::Subtract, lfcalc::typed_subtract([](const lfcalc::SubtractReduction& ctx) -> double {
            return ctx.left - ctx.right;
        })},
        {lfcalc::SemanticAction::Multiply, lfcalc::typed_multiply([](const lfcalc::MultiplyReduction& ctx) -> double {
            return ctx.left * ctx.right;
        })},
        {lfcalc::SemanticAction::Divide, lfcalc::typed_divide([](const lfcalc::DivideReduction& ctx) -> double {
            if (ctx.right == 0.0) {
                throw std::runtime_error("division by zero");
            }
            return ctx.left / ctx.right;
        })},
    };
    return reducers;
}

/// Builds reducers through generated typed contexts while reusing boxed semantics.
///
/// This is the migration bridge for older boxed reducers. It validates named
/// contexts first, then delegates to `make_boxed_reducers`.
static const lfcalc::ReducerMap& make_boxed_to_typed_reducers() {
    static const lfcalc::ReducerMap reducers = lfcalc::typed_reducer_map_from_boxed(make_boxed_reducers());
    return reducers;
}

/// Selects the reducer ABI used by the demo.
static const lfcalc::ReducerMap& make_reducers(ReducerMode mode) {
    switch (mode) {
    case ReducerMode::DirectTyped:
        return make_direct_typed_reducers();
    case ReducerMode::BoxedToTyped:
        return make_boxed_to_typed_reducers();
    case ReducerMode::Boxed:
        return make_boxed_reducers();
    }
    throw std::runtime_error("unknown reducer mode");
}

/// Scans, parses, and evaluates one calculator expression from a stream.
///
/// InputStreamScanner pulls UTF-8 source text only when the parser asks for the next
/// lexeme. Returned lexeme text is owned by the scanner, so the scanner must
/// stay alive until parsing and reducer code finish.
static double evaluate_stream(std::istream& source, ReducerMode mode = ReducerMode::DirectTyped, std::size_t read_buffer_size = 4096, std::size_t max_buffered_lexeme_bytes = 1048576) {
    lfcalc::InputStreamScanner scanner(source, read_buffer_size, max_buffered_lexeme_bytes);
    const auto value = lfcalc::parse_value(scanner, make_reducers(mode));
    return std::any_cast<double>(value);
}

/// Scans, parses, and evaluates one calculator expression from in-memory text.
static double evaluate(std::string_view source, ReducerMode mode = ReducerMode::DirectTyped) {
    std::istringstream input{std::string(source)};
    return evaluate_stream(input, mode);
}

/// Throws when a condition used by the example self-test is false.
static void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

/// Covers behavior that is easy to regress while changing generated runtimes.
static void run_assertions() {
    require(std::fabs(evaluate("1 + 2 * (3 - 4.5)") - -2.0) < 0.000001, "wrong expression result");
    {
        std::istringstream input{"1 + 2 * (3 - 4.5)"};
        require(std::fabs(evaluate_stream(input, ReducerMode::DirectTyped, 1) - -2.0) < 0.000001, "wrong chunked stream expression result");
    }
    require(std::fabs(evaluate("7.5/2.5", ReducerMode::Boxed) - 3.0) < 0.000001, "wrong boxed decimal division result");
    require(std::fabs(evaluate("3+4", ReducerMode::BoxedToTyped) - 7.0) < 0.000001, "wrong boxed-to-typed migration result");

    const auto visible = lfcalc::tokenize("1+2");
    lfcalc::parse(visible);

    auto with_eof = visible;
    with_eof.push_back(lfcalc::Lexeme{lfcalc::Token::End, "", "", 0, 0, 1, 1, 1, 1});
    lfcalc::parse(with_eof);

    with_eof.push_back(lfcalc::Lexeme{lfcalc::Token::Plus, "+", "", 0, 1, 1, 1, 1, 2});
    try {
        lfcalc::parse(with_eof);
        throw std::runtime_error("expected token-after-EOF parse error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("token after EOF") != std::string::npos, "wrong EOF error");
    }

    try {
        lfcalc::tokenize("1@");
        throw std::runtime_error("expected scanner error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("no lexical rule") != std::string::npos, "wrong scanner error");
    }

    try {
        evaluate("1@");
        throw std::runtime_error("expected source scanner error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("no lexical rule") != std::string::npos, "wrong source scanner error");
    }

    try {
        evaluate("1+");
        throw std::runtime_error("expected source parser error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("parse error") != std::string::npos, "wrong source parse error");
    }

    try {
        std::istringstream input{"123"};
        evaluate_stream(input, ReducerMode::DirectTyped, 1, 2);
        throw std::runtime_error("expected buffered-lexeme stream error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("buffered lexeme exceeds") != std::string::npos, "wrong stream buffer-limit error");
    }

    try {
        evaluate("1/0");
        throw std::runtime_error("expected division-by-zero error");
    } catch (const std::runtime_error& ex) {
        require(std::string(ex.what()).find("division by zero") != std::string::npos, "wrong divide error");
    }

    lfcalc::Parser parser;
    std::vector<std::thread> workers;
    for (int i = 0; i < 8; ++i) {
        workers.emplace_back([&parser]() {
            lfcalc::Scanner scanner("1 + 2 * (3 - 4.5)");
            parser.parse(scanner);
        });
    }
    for (auto& worker : workers) {
        worker.join();
    }

    lfcalc::Scanner shared("1 + 2 * (3 - 4.5)");
    std::atomic<int> count{0};
    workers.clear();
    for (int i = 0; i < 4; ++i) {
        workers.emplace_back([&shared, &count]() {
            lfcalc::Lexeme lexeme;
            while (shared.next(lexeme)) {
                ++count;
            }
        });
    }
    for (auto& worker : workers) {
        worker.join();
    }
    require(count == static_cast<int>(lfcalc::tokenize("1 + 2 * (3 - 4.5)").size()), "shared scanner token count mismatch");
}

/// Removes an option with one value from argv-like storage.
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

/// Removes a flag from argv-like storage.
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
        const bool boxed_typed_mode = take_flag(args, "--boxed-typed");
        const ReducerMode mode = boxed_typed_mode ? ReducerMode::BoxedToTyped : (boxed_mode ? ReducerMode::Boxed : ReducerMode::DirectTyped);
        const std::string log_path = read_option(args, "--log", "dist/calc-cpp-demo.log");
        const std::string input_path = args.empty() ? "input.calc" : args.front();

        if (assert_mode) {
            run_assertions();
        }

        const std::string source = read_text_file(input_path);
        std::ifstream input(input_path);
        if (!input) {
            throw std::runtime_error("cannot open input file for parse: " + input_path);
        }
        const double result = evaluate_stream(input, mode);
        std::ostringstream report;
        report << "LangForge C++ calculator demo\n";
        report << "source: " << source;
        if (source.empty() || source.back() != '\n') {
            report << '\n';
        }
        report << "result: " << std::setprecision(10) << result << '\n';
        write_text_file(log_path, report.str());
        std::cout << report.str();
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << '\n';
        return 1;
    }
}
