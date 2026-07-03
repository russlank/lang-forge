#pragma once

#include "mini/ast.hpp"

#include <string>
#include <string_view>
#include <vector>

namespace mini::compiler {

enum class OpCode {
    Push,
    Add,
    Print,
};

struct Instruction {
    OpCode op = OpCode::Push;
    int argument = 0;
};

/// Lowers the AST into a tiny stack-machine instruction sequence.
std::vector<Instruction> compile(const ast::Program& program);

/// Executes stack-machine instructions and returns printed values.
std::vector<int> execute(const std::vector<Instruction>& code);

/// Formats the compiler and runtime output for the demo CLI.
std::string format_report(std::string_view input_path, const std::vector<Instruction>& code, const std::vector<int>& output);

} // namespace mini::compiler
