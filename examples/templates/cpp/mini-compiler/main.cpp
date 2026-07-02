#include "generated/parser.hpp"

#include <any>
#include <fstream>
#include <iostream>
#include <memory>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <vector>

namespace mini = LangForge::Examples::Templates::MiniCompiler::Generated;

struct Expr;
using ExprPtr = std::shared_ptr<Expr>;

struct Expr {
    enum class Kind { Number, Add };
    Kind kind;
    int value = 0;
    ExprPtr left;
    ExprPtr right;
};

struct Statement {
    ExprPtr expr;
};

struct Program {
    std::vector<Statement> statements;
};

struct Instruction {
    std::string op;
    int arg = 0;
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

template <typename T>
static T arg(const mini::Reduction& ctx, std::size_t index, std::string_view name) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing " + std::string(name));
    }
    return std::any_cast<T>(ctx.values.at(index));
}

static std::string text(const mini::Reduction& ctx, std::size_t index, std::string_view name) {
    return std::string(arg<mini::Lexeme>(ctx, index, name).text);
}

static ExprPtr number_expr(int value) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Number;
    expr->value = value;
    return expr;
}

static ExprPtr add_expr(ExprPtr left, ExprPtr right) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Add;
    expr->left = std::move(left);
    expr->right = std::move(right);
    return expr;
}

static std::vector<Statement> prepend(Statement head, std::vector<Statement> tail) {
    std::vector<Statement> out;
    out.reserve(tail.size() + 1);
    out.push_back(std::move(head));
    out.insert(out.end(), tail.begin(), tail.end());
    return out;
}

static mini::ReducerMap reducers() {
    return mini::ReducerMap{
        {mini::SemanticAction::Program, [](const mini::Reduction& ctx) -> mini::Value {
            return Program{arg<std::vector<Statement>>(ctx, 0, "statements")};
        }},
        {mini::SemanticAction::Statements, [](const mini::Reduction& ctx) -> mini::Value {
            return prepend(arg<Statement>(ctx, 0, "statement"), arg<std::vector<Statement>>(ctx, 1, "statement tail"));
        }},
        {mini::SemanticAction::StatementsTailMore, [](const mini::Reduction& ctx) -> mini::Value {
            return prepend(arg<Statement>(ctx, 0, "statement"), arg<std::vector<Statement>>(ctx, 1, "statement tail"));
        }},
        {mini::SemanticAction::StatementsTailEmpty, [](const mini::Reduction&) -> mini::Value {
            return std::vector<Statement>{};
        }},
        {mini::SemanticAction::Print, [](const mini::Reduction& ctx) -> mini::Value {
            return Statement{arg<ExprPtr>(ctx, 1, "print expression")};
        }},
        {mini::SemanticAction::Add, [](const mini::Reduction& ctx) -> mini::Value {
            return add_expr(arg<ExprPtr>(ctx, 0, "left operand"), arg<ExprPtr>(ctx, 2, "right operand"));
        }},
        {mini::SemanticAction::Pass, [](const mini::Reduction& ctx) -> mini::Value {
            return ctx.values.at(0);
        }},
        {mini::SemanticAction::Number, [](const mini::Reduction& ctx) -> mini::Value {
            return number_expr(std::stoi(text(ctx, 0, "number literal")));
        }},
    };
}

static Program parse_program(std::string_view source) {
    mini::Scanner scanner(source);
    return std::any_cast<Program>(mini::parse_value(scanner, reducers()));
}

static void compile_expr(const ExprPtr& expr, std::vector<Instruction>& code) {
    if (expr->kind == Expr::Kind::Number) {
        code.push_back({"push", expr->value});
        return;
    }
    compile_expr(expr->left, code);
    compile_expr(expr->right, code);
    code.push_back({"add", 0});
}

static std::vector<Instruction> compile_program(const Program& program) {
    std::vector<Instruction> code;
    for (const auto& statement : program.statements) {
        compile_expr(statement.expr, code);
        code.push_back({"print", 0});
    }
    return code;
}

static std::vector<int> run(const std::vector<Instruction>& code) {
    std::vector<int> stack;
    std::vector<int> output;
    for (std::size_t pc = 0; pc < code.size(); ++pc) {
        const auto& instruction = code[pc];
        if (instruction.op == "push") {
            stack.push_back(instruction.arg);
        } else if (instruction.op == "add") {
            if (stack.size() < 2) {
                throw std::runtime_error("pc " + std::to_string(pc) + ": add needs two stack values");
            }
            const int right = stack.back();
            stack.pop_back();
            const int left = stack.back();
            stack.back() = left + right;
        } else if (instruction.op == "print") {
            if (stack.empty()) {
                throw std::runtime_error("pc " + std::to_string(pc) + ": print needs one stack value");
            }
            output.push_back(stack.back());
            stack.pop_back();
        }
    }
    return output;
}

static std::string report(const std::string& input_path, const std::vector<Instruction>& code, const std::vector<int>& output) {
    std::ostringstream out;
    out << "Mini compiler C++ template: " << input_path << "\nstack code:\n";
    for (std::size_t i = 0; i < code.size(); ++i) {
        out << "  ";
        if (i < 10) {
            out << "0";
        }
        out << i << " " << code[i].op;
        if (code[i].op == "push") {
            out << " " << code[i].arg;
        }
        out << "\n";
    }
    out << "output:";
    for (int value : output) {
        out << " " << value;
    }
    out << "\n";
    return out.str();
}

static void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

static void run_assertions(const std::string& source) {
    const auto output = run(compile_program(parse_program(source)));
    require(output.size() == 2 && output[0] == 3 && output[1] == 42, "unexpected template output");
    try {
        parse_program("print 1 +;");
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
        const std::string log_path = read_option(args, "--log", "dist/mini-cpp.log");
        const std::string input_path = args.empty() ? "input.mini" : args.front();
        const std::string source = read_text_file(input_path);
        if (assert_mode) {
            run_assertions(source);
        }
        const auto code = compile_program(parse_program(source));
        const auto output = run(code);
        const auto text = report(input_path, code, output);
        std::cout << text;
        write_text_file(log_path, text);
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
