#include "mini/compiler.hpp"

#include <sstream>
#include <stdexcept>

namespace mini::compiler {

namespace {

template <typename... Ts>
struct Overloaded : Ts... {
    using Ts::operator()...;
};

template <typename... Ts>
Overloaded(Ts...) -> Overloaded<Ts...>;

void compile_expr(const ast::Program& program, ast::ExprId id, std::vector<Instruction>& code) {
    const auto& expr = program.expression(id);
    std::visit(Overloaded{
                   [&](const ast::NumberExpr& number) {
                       code.push_back({OpCode::Push, number.value});
                   },
                   [&](const ast::AddExpr& add) {
                       compile_expr(program, add.left, code);
                       compile_expr(program, add.right, code);
                       code.push_back({OpCode::Add, 0});
                   },
               },
        expr.node);
}

std::string op_name(OpCode op) {
    switch (op) {
    case OpCode::Push:
        return "push";
    case OpCode::Add:
        return "add";
    case OpCode::Print:
        return "print";
    }
    return "unknown";
}

} // namespace

std::vector<Instruction> compile(const ast::Program& program) {
    std::vector<Instruction> code;
    for (const auto& statement : program.statements) {
        std::visit(Overloaded{
                       [&](const ast::PrintStatement& print) {
                           compile_expr(program, print.expr, code);
                           code.push_back({OpCode::Print, 0});
                       },
                   },
            statement);
    }
    return code;
}

std::vector<int> execute(const std::vector<Instruction>& code) {
    std::vector<int> stack;
    std::vector<int> output;
    for (std::size_t pc = 0; pc < code.size(); ++pc) {
        const auto& instruction = code[pc];
        switch (instruction.op) {
        case OpCode::Push:
            stack.push_back(instruction.argument);
            break;
        case OpCode::Add: {
            if (stack.size() < 2) {
                throw std::runtime_error("pc " + std::to_string(pc) + ": add needs two stack values");
            }
            const int right = stack.back();
            stack.pop_back();
            const int left = stack.back();
            stack.back() = left + right;
            break;
        }
        case OpCode::Print:
            if (stack.empty()) {
                throw std::runtime_error("pc " + std::to_string(pc) + ": print needs one stack value");
            }
            output.push_back(stack.back());
            stack.pop_back();
            break;
        }
    }
    return output;
}

std::string format_report(std::string_view input_path, const std::vector<Instruction>& code, const std::vector<int>& output) {
    std::ostringstream out;
    out << "Layered C++ compiler template: " << input_path << "\nstack code:\n";
    for (std::size_t i = 0; i < code.size(); ++i) {
        out << "  ";
        if (i < 10) {
            out << "0";
        }
        out << i << " " << op_name(code[i].op);
        if (code[i].op == OpCode::Push) {
            out << " " << code[i].argument;
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

} // namespace mini::compiler
