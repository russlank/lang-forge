#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <utility>
#include <vector>

namespace lfdraw {

/// RGB color used by the renderer and PNG writer.
struct Color {
    std::uint8_t r = 0;
    std::uint8_t g = 0;
    std::uint8_t b = 0;
};

struct Expr;
struct Statement;
struct FigureRef;
struct FigureBlock;

using ExprPtr = std::shared_ptr<Expr>;
using StatementPtr = std::shared_ptr<Statement>;
using FigureRefPtr = std::shared_ptr<FigureRef>;
using FigureBlockPtr = std::shared_ptr<FigureBlock>;
using StatementList = std::vector<StatementPtr>;
using TailList = std::vector<std::pair<char, ExprPtr>>;

/// Numeric expression node built by generated parser reductions.
struct Expr {
    enum class Kind { Number, Variable, Unary, Binary, Call };
    Kind kind = Kind::Number;
    double number = 0.0;
    std::string name;
    char op = 0;
    ExprPtr left;
    ExprPtr right;
    ExprPtr arg;
};

/// Reusable figure-local statement list.
struct FigureBlock {
    StatementList statements;
};

/// Reference to a named or inline figure.
struct FigureRef {
    bool named = false;
    std::string name;
    FigureBlockPtr block;
};

/// Executable DRAW statement node.
struct Statement {
    enum class Kind { Canvas, Background, Stroke, Fill, Width, Assign, DefineFigure, Draw, Repdraw, Primitive };
    Kind kind = Kind::Canvas;
    std::string name;
    std::string primitive;
    Color color{};
    bool enabled = false;
    std::vector<ExprPtr> exprs;
    FigureBlockPtr figure;
    FigureRefPtr target;
};

/// Root node for a DRAW source file.
struct Program {
    StatementList statements;
};

using ProgramPtr = std::shared_ptr<Program>;

} // namespace lfdraw
