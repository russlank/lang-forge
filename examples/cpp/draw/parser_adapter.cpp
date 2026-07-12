#include "parser_adapter.hpp"

#include "generated/parser_typed.hpp"

#include <algorithm>
#include <any>
#include <cmath>
#include <iomanip>
#include <sstream>
#include <stdexcept>

namespace draw = LangForge::Examples::Draw::Generated;

namespace lfdraw {

static draw::Lexeme lexeme_arg(const draw::Reduction& ctx, std::size_t index, const std::string& name) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing " + name);
    }
    try {
        return std::any_cast<draw::Lexeme>(ctx.values.at(index));
    } catch (const std::bad_any_cast&) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " argument " + std::to_string(index + 1) + " is not a lexeme");
    }
}

static std::string text_arg(const draw::Reduction& ctx, std::size_t index, const std::string& name) {
    return std::string(lexeme_arg(ctx, index, name).text);
}

template <typename T>
static T value_arg(const draw::Reduction& ctx, std::size_t index, const std::string& name) {
    // Boxed reducers still receive std::any values. Typed mode validates the
    // generated named-label context first, then delegates to this boxed
    // implementation so AST construction remains a single source of truth.
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing " + name);
    }
    try {
        return std::any_cast<T>(ctx.values.at(index));
    } catch (const std::bad_any_cast&) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " argument " + std::to_string(index + 1) + " has unexpected type for " + name);
    }
}

static ExprPtr number_expr(double value) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Number;
    expr->number = value;
    return expr;
}

static ExprPtr variable_expr(std::string name) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Variable;
    expr->name = std::move(name);
    return expr;
}

static ExprPtr unary_expr(char op, ExprPtr arg) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Unary;
    expr->op = op;
    expr->arg = std::move(arg);
    return expr;
}

static ExprPtr binary_expr(char op, ExprPtr left, ExprPtr right) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Binary;
    expr->op = op;
    expr->left = std::move(left);
    expr->right = std::move(right);
    return expr;
}

static ExprPtr call_expr(std::string name, ExprPtr arg) {
    auto expr = std::make_shared<Expr>();
    expr->kind = Expr::Kind::Call;
    expr->name = std::move(name);
    expr->arg = std::move(arg);
    return expr;
}

static StatementPtr statement(Statement::Kind kind) {
    auto item = std::make_shared<Statement>();
    item->kind = kind;
    return item;
}

static StatementPtr primitive_statement(std::string kind, std::vector<ExprPtr> args) {
    auto item = statement(Statement::Kind::Primitive);
    item->primitive = std::move(kind);
    item->exprs = std::move(args);
    return item;
}

static StatementList prepend(StatementPtr statement, StatementList tail) {
    StatementList result;
    result.reserve(tail.size() + 1);
    result.push_back(std::move(statement));
    result.insert(result.end(), tail.begin(), tail.end());
    return result;
}

static TailList prepend_tail(char op, ExprPtr right, TailList tail) {
    TailList result;
    result.reserve(tail.size() + 1);
    result.emplace_back(op, std::move(right));
    result.insert(result.end(), tail.begin(), tail.end());
    return result;
}

static ExprPtr fold_binary(ExprPtr left, const TailList& tails) {
    ExprPtr result = std::move(left);
    for (const auto& tail : tails) {
        result = binary_expr(tail.first, result, tail.second);
    }
    return result;
}

static int hex(char ch) {
    if (ch >= '0' && ch <= '9') {
        return ch - '0';
    }
    if (ch >= 'a' && ch <= 'f') {
        return ch - 'a' + 10;
    }
    if (ch >= 'A' && ch <= 'F') {
        return ch - 'A' + 10;
    }
    throw std::runtime_error("invalid color digit");
}

static Color parse_color(std::string text) {
    if (text.size() != 7 || text[0] != '#') {
        throw std::runtime_error("invalid color: " + text);
    }
    return Color{
        static_cast<std::uint8_t>(hex(text[1]) * 16 + hex(text[2])),
        static_cast<std::uint8_t>(hex(text[3]) * 16 + hex(text[4])),
        static_cast<std::uint8_t>(hex(text[5]) * 16 + hex(text[6])),
    };
}

const draw::ReducerMap& make_reducers() {
    // The map keys are generated from {cpp: ...} labels in draw.lf. Each lambda
    // is handwritten semantic code that builds the DRAW AST consumed by the
    // renderer.
    static const draw::ReducerMap reducers{
        {draw::SemanticAction::Program, [](const draw::Reduction& ctx) -> draw::Value {
            auto program = std::make_shared<Program>();
            program->statements = value_arg<StatementList>(ctx, 0, "statement list");
            return program;
        }},
        {draw::SemanticAction::Statements, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 0, "statement"), value_arg<StatementList>(ctx, 1, "tail statements"));
        }},
        {draw::SemanticAction::Figures, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 0, "figure statement"), value_arg<StatementList>(ctx, 1, "tail figures"));
        }},
        {draw::SemanticAction::StatementTailMore, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 1, "statement"), value_arg<StatementList>(ctx, 2, "tail statements"));
        }},
        {draw::SemanticAction::FigureTailMore, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 1, "figure statement"), value_arg<StatementList>(ctx, 2, "tail figures"));
        }},
        {draw::SemanticAction::StatementTailEmpty, [](const draw::Reduction&) -> draw::Value { return StatementList{}; }},
        {draw::SemanticAction::FigureTailEmpty, [](const draw::Reduction&) -> draw::Value { return StatementList{}; }},
        {draw::SemanticAction::Pass, [](const draw::Reduction& ctx) -> draw::Value { return ctx.values.at(0); }},
        {draw::SemanticAction::Canvas, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Canvas);
            item->exprs = {value_arg<ExprPtr>(ctx, 1, "width"), value_arg<ExprPtr>(ctx, 3, "height")};
            return item;
        }},
        {draw::SemanticAction::Background, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Background);
            item->color = value_arg<Color>(ctx, 1, "background color");
            return item;
        }},
        {draw::SemanticAction::Stroke, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Stroke);
            item->color = value_arg<Color>(ctx, 1, "stroke color");
            return item;
        }},
        {draw::SemanticAction::Fill, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Fill);
            item->color = value_arg<Color>(ctx, 1, "fill color");
            item->enabled = true;
            return item;
        }},
        {draw::SemanticAction::FillNone, [](const draw::Reduction&) -> draw::Value {
            auto item = statement(Statement::Kind::Fill);
            item->enabled = false;
            return item;
        }},
        {draw::SemanticAction::Width, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Width);
            item->exprs = {value_arg<ExprPtr>(ctx, 1, "line width")};
            return item;
        }},
        {draw::SemanticAction::Assign, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Assign);
            item->name = text_arg(ctx, 0, "variable name");
            item->exprs = {value_arg<ExprPtr>(ctx, 2, "assigned value")};
            return item;
        }},
        {draw::SemanticAction::DefineFigure, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::DefineFigure);
            item->name = text_arg(ctx, 0, "figure name");
            item->figure = value_arg<FigureBlockPtr>(ctx, 2, "figure block");
            return item;
        }},
        {draw::SemanticAction::Draw, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Draw);
            item->target = value_arg<FigureRefPtr>(ctx, 1, "figure reference");
            return item;
        }},
        {draw::SemanticAction::Repdraw, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Repdraw);
            item->exprs = {value_arg<ExprPtr>(ctx, 1, "repeat count")};
            item->target = value_arg<FigureRefPtr>(ctx, 2, "figure reference");
            return item;
        }},
        {draw::SemanticAction::FigureRefNamed, [](const draw::Reduction& ctx) -> draw::Value {
            auto ref = std::make_shared<FigureRef>();
            ref->named = true;
            ref->name = text_arg(ctx, 0, "figure name");
            return ref;
        }},
        {draw::SemanticAction::FigureRefInline, [](const draw::Reduction& ctx) -> draw::Value {
            auto ref = std::make_shared<FigureRef>();
            ref->block = value_arg<FigureBlockPtr>(ctx, 0, "inline figure");
            return ref;
        }},
        {draw::SemanticAction::FigureBlock, [](const draw::Reduction& ctx) -> draw::Value {
            auto block = std::make_shared<FigureBlock>();
            block->statements = value_arg<StatementList>(ctx, 1, "figure statements");
            return block;
        }},
        {draw::SemanticAction::PrimitivePoint, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("point", {value_arg<ExprPtr>(ctx, 1, "x"), value_arg<ExprPtr>(ctx, 3, "y")});
        }},
        {draw::SemanticAction::PrimitiveLine, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("line", {value_arg<ExprPtr>(ctx, 1, "x1"), value_arg<ExprPtr>(ctx, 3, "y1"), value_arg<ExprPtr>(ctx, 5, "x2"), value_arg<ExprPtr>(ctx, 7, "y2")});
        }},
        {draw::SemanticAction::PrimitiveBox, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("box", {value_arg<ExprPtr>(ctx, 1, "x1"), value_arg<ExprPtr>(ctx, 3, "y1"), value_arg<ExprPtr>(ctx, 5, "x2"), value_arg<ExprPtr>(ctx, 7, "y2")});
        }},
        {draw::SemanticAction::PrimitiveCircle, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("circle", {value_arg<ExprPtr>(ctx, 1, "cx"), value_arg<ExprPtr>(ctx, 3, "cy"), value_arg<ExprPtr>(ctx, 5, "radius")});
        }},
        {draw::SemanticAction::Color, [](const draw::Reduction& ctx) -> draw::Value { return parse_color(text_arg(ctx, 0, "color literal")); }},
        {draw::SemanticAction::Expr, [](const draw::Reduction& ctx) -> draw::Value {
            return fold_binary(value_arg<ExprPtr>(ctx, 0, "left expression"), value_arg<TailList>(ctx, 1, "expression tail"));
        }},
        {draw::SemanticAction::Term, [](const draw::Reduction& ctx) -> draw::Value {
            return fold_binary(value_arg<ExprPtr>(ctx, 0, "left term"), value_arg<TailList>(ctx, 1, "term tail"));
        }},
        {draw::SemanticAction::ExprTailAdd, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('+', value_arg<ExprPtr>(ctx, 1, "right expression"), value_arg<TailList>(ctx, 2, "tail expressions"));
        }},
        {draw::SemanticAction::ExprTailSubtract, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('-', value_arg<ExprPtr>(ctx, 1, "right expression"), value_arg<TailList>(ctx, 2, "tail expressions"));
        }},
        {draw::SemanticAction::TermTailMultiply, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('*', value_arg<ExprPtr>(ctx, 1, "right term"), value_arg<TailList>(ctx, 2, "tail terms"));
        }},
        {draw::SemanticAction::TermTailDivide, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('/', value_arg<ExprPtr>(ctx, 1, "right term"), value_arg<TailList>(ctx, 2, "tail terms"));
        }},
        {draw::SemanticAction::ExprTailEmpty, [](const draw::Reduction&) -> draw::Value { return TailList{}; }},
        {draw::SemanticAction::TermTailEmpty, [](const draw::Reduction&) -> draw::Value { return TailList{}; }},
        {draw::SemanticAction::UnaryNegate, [](const draw::Reduction& ctx) -> draw::Value {
            return unary_expr('-', value_arg<ExprPtr>(ctx, 1, "operand"));
        }},
        {draw::SemanticAction::ExprPass, [](const draw::Reduction& ctx) -> draw::Value {
            return value_arg<ExprPtr>(ctx, 0, "expression");
        }},
        {draw::SemanticAction::Number, [](const draw::Reduction& ctx) -> draw::Value {
            return number_expr(std::stod(text_arg(ctx, 0, "number")));
        }},
        {draw::SemanticAction::Variable, [](const draw::Reduction& ctx) -> draw::Value {
            return variable_expr(text_arg(ctx, 0, "variable name"));
        }},
        {draw::SemanticAction::Call, [](const draw::Reduction& ctx) -> draw::Value {
            return call_expr(text_arg(ctx, 0, "function name"), value_arg<ExprPtr>(ctx, 2, "argument"));
        }},
        {draw::SemanticAction::Group, [](const draw::Reduction& ctx) -> draw::Value { return ctx.values.at(1); }},
    };
    return reducers;
}

const draw::ReducerMap& make_typed_reducers() {
    static const draw::ReducerMap reducers = draw::typed_reducer_map_from_boxed(make_reducers());
    return reducers;
}

ProgramPtr parse_program(const std::string& source, bool typed) {
    draw::Scanner scanner(source);
    return std::any_cast<ProgramPtr>(draw::parse_value(scanner, typed ? make_typed_reducers() : make_reducers()));
}

} // namespace lfdraw
