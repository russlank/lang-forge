#include "generated/parser.hpp"

#include <algorithm>
#include <array>
#include <any>
#include <cmath>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <map>
#include <memory>
#include <sstream>
#include <stdexcept>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>
#include <vector>

namespace draw = LangForge::Examples::Draw::Generated;

constexpr int max_repdraw_iterations = 20000;

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

struct FigureBlock {
    StatementList statements;
};

struct FigureRef {
    bool named = false;
    std::string name;
    FigureBlockPtr block;
};

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

struct Program {
    StatementList statements;
};

using ProgramPtr = std::shared_ptr<Program>;
using TailList = std::vector<std::pair<char, ExprPtr>>;

static std::string read_text_file(const std::string& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open input file: " + path);
    }
    std::ostringstream buffer;
    buffer << input.rdbuf();
    return buffer.str();
}

static void ensure_parent_dir(const std::string& path) {
    const std::filesystem::path parent = std::filesystem::path(path).parent_path();
    if (parent.empty()) {
        return;
    }
    std::error_code ec;
    std::filesystem::create_directories(parent, ec);
    if (ec) {
        throw std::runtime_error("cannot create output directory: " + parent.string());
    }
}

static void write_text_file(const std::string& path, std::string_view text) {
    ensure_parent_dir(path);
    std::ofstream output(path);
    if (!output) {
        throw std::runtime_error("cannot open log file: " + path);
    }
    output << text;
}

static draw::Lexeme lexeme_arg(const draw::Reduction& ctx, std::size_t index) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing lexeme argument");
    }
    return std::any_cast<draw::Lexeme>(ctx.values.at(index));
}

static std::string text_arg(const draw::Reduction& ctx, std::size_t index) {
    return std::string(lexeme_arg(ctx, index).text);
}

template <typename T>
static T value_arg(const draw::Reduction& ctx, std::size_t index) {
    if (index >= ctx.values.size()) {
        throw std::runtime_error("rule " + std::to_string(ctx.rule) + " missing semantic argument");
    }
    return std::any_cast<T>(ctx.values.at(index));
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

static std::string color_hex(Color color) {
    std::ostringstream out;
    out << "#" << std::uppercase << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(color.r)
        << std::setw(2) << static_cast<int>(color.g)
        << std::setw(2) << static_cast<int>(color.b);
    return out.str();
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

static draw::ReducerMap make_reducers() {
    return draw::ReducerMap{
        {draw::SemanticAction::Program, [](const draw::Reduction& ctx) -> draw::Value {
            auto program = std::make_shared<Program>();
            program->statements = value_arg<StatementList>(ctx, 0);
            return program;
        }},
        {draw::SemanticAction::Statements, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 0), value_arg<StatementList>(ctx, 1));
        }},
        {draw::SemanticAction::Figures, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 0), value_arg<StatementList>(ctx, 1));
        }},
        {draw::SemanticAction::StatementTailMore, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 1), value_arg<StatementList>(ctx, 2));
        }},
        {draw::SemanticAction::FigureTailMore, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend(value_arg<StatementPtr>(ctx, 1), value_arg<StatementList>(ctx, 2));
        }},
        {draw::SemanticAction::StatementTailEmpty, [](const draw::Reduction&) -> draw::Value { return StatementList{}; }},
        {draw::SemanticAction::FigureTailEmpty, [](const draw::Reduction&) -> draw::Value { return StatementList{}; }},
        {draw::SemanticAction::Pass, [](const draw::Reduction& ctx) -> draw::Value { return ctx.values.at(0); }},
        {draw::SemanticAction::Canvas, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Canvas);
            item->exprs = {value_arg<ExprPtr>(ctx, 1), value_arg<ExprPtr>(ctx, 3)};
            return item;
        }},
        {draw::SemanticAction::Background, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Background);
            item->color = value_arg<Color>(ctx, 1);
            return item;
        }},
        {draw::SemanticAction::Stroke, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Stroke);
            item->color = value_arg<Color>(ctx, 1);
            return item;
        }},
        {draw::SemanticAction::Fill, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Fill);
            item->color = value_arg<Color>(ctx, 1);
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
            item->exprs = {value_arg<ExprPtr>(ctx, 1)};
            return item;
        }},
        {draw::SemanticAction::Assign, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Assign);
            item->name = text_arg(ctx, 0);
            item->exprs = {value_arg<ExprPtr>(ctx, 2)};
            return item;
        }},
        {draw::SemanticAction::DefineFigure, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::DefineFigure);
            item->name = text_arg(ctx, 0);
            item->figure = value_arg<FigureBlockPtr>(ctx, 2);
            return item;
        }},
        {draw::SemanticAction::Draw, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Draw);
            item->target = value_arg<FigureRefPtr>(ctx, 1);
            return item;
        }},
        {draw::SemanticAction::Repdraw, [](const draw::Reduction& ctx) -> draw::Value {
            auto item = statement(Statement::Kind::Repdraw);
            item->exprs = {value_arg<ExprPtr>(ctx, 1)};
            item->target = value_arg<FigureRefPtr>(ctx, 2);
            return item;
        }},
        {draw::SemanticAction::FigureRefNamed, [](const draw::Reduction& ctx) -> draw::Value {
            auto ref = std::make_shared<FigureRef>();
            ref->named = true;
            ref->name = text_arg(ctx, 0);
            return ref;
        }},
        {draw::SemanticAction::FigureRefInline, [](const draw::Reduction& ctx) -> draw::Value {
            auto ref = std::make_shared<FigureRef>();
            ref->block = value_arg<FigureBlockPtr>(ctx, 0);
            return ref;
        }},
        {draw::SemanticAction::FigureBlock, [](const draw::Reduction& ctx) -> draw::Value {
            auto block = std::make_shared<FigureBlock>();
            block->statements = value_arg<StatementList>(ctx, 1);
            return block;
        }},
        {draw::SemanticAction::PrimitivePoint, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("point", {value_arg<ExprPtr>(ctx, 1), value_arg<ExprPtr>(ctx, 3)});
        }},
        {draw::SemanticAction::PrimitiveLine, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("line", {value_arg<ExprPtr>(ctx, 1), value_arg<ExprPtr>(ctx, 3), value_arg<ExprPtr>(ctx, 5), value_arg<ExprPtr>(ctx, 7)});
        }},
        {draw::SemanticAction::PrimitiveBox, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("box", {value_arg<ExprPtr>(ctx, 1), value_arg<ExprPtr>(ctx, 3), value_arg<ExprPtr>(ctx, 5), value_arg<ExprPtr>(ctx, 7)});
        }},
        {draw::SemanticAction::PrimitiveCircle, [](const draw::Reduction& ctx) -> draw::Value {
            return primitive_statement("circle", {value_arg<ExprPtr>(ctx, 1), value_arg<ExprPtr>(ctx, 3), value_arg<ExprPtr>(ctx, 5)});
        }},
        {draw::SemanticAction::Color, [](const draw::Reduction& ctx) -> draw::Value { return parse_color(text_arg(ctx, 0)); }},
        {draw::SemanticAction::Expr, [](const draw::Reduction& ctx) -> draw::Value {
            return fold_binary(value_arg<ExprPtr>(ctx, 0), value_arg<TailList>(ctx, 1));
        }},
        {draw::SemanticAction::Term, [](const draw::Reduction& ctx) -> draw::Value {
            return fold_binary(value_arg<ExprPtr>(ctx, 0), value_arg<TailList>(ctx, 1));
        }},
        {draw::SemanticAction::ExprTailAdd, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('+', value_arg<ExprPtr>(ctx, 1), value_arg<TailList>(ctx, 2));
        }},
        {draw::SemanticAction::ExprTailSubtract, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('-', value_arg<ExprPtr>(ctx, 1), value_arg<TailList>(ctx, 2));
        }},
        {draw::SemanticAction::TermTailMultiply, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('*', value_arg<ExprPtr>(ctx, 1), value_arg<TailList>(ctx, 2));
        }},
        {draw::SemanticAction::TermTailDivide, [](const draw::Reduction& ctx) -> draw::Value {
            return prepend_tail('/', value_arg<ExprPtr>(ctx, 1), value_arg<TailList>(ctx, 2));
        }},
        {draw::SemanticAction::ExprTailEmpty, [](const draw::Reduction&) -> draw::Value { return TailList{}; }},
        {draw::SemanticAction::TermTailEmpty, [](const draw::Reduction&) -> draw::Value { return TailList{}; }},
        {draw::SemanticAction::UnaryNegate, [](const draw::Reduction& ctx) -> draw::Value {
            return unary_expr('-', value_arg<ExprPtr>(ctx, 1));
        }},
        {draw::SemanticAction::Number, [](const draw::Reduction& ctx) -> draw::Value {
            return number_expr(std::stod(text_arg(ctx, 0)));
        }},
        {draw::SemanticAction::Variable, [](const draw::Reduction& ctx) -> draw::Value {
            return variable_expr(text_arg(ctx, 0));
        }},
        {draw::SemanticAction::Call, [](const draw::Reduction& ctx) -> draw::Value {
            return call_expr(text_arg(ctx, 0), value_arg<ExprPtr>(ctx, 2));
        }},
        {draw::SemanticAction::Group, [](const draw::Reduction& ctx) -> draw::Value { return ctx.values.at(1); }},
    };
}

struct Image {
    int width = 0;
    int height = 0;
    std::vector<Color> pixels;

    void reset(int w, int h) {
        if (w <= 0 || h <= 0 || w > 4096 || h > 4096) {
            throw std::runtime_error("canvas dimensions must be in 1..4096");
        }
        width = w;
        height = h;
        pixels.assign(static_cast<std::size_t>(w * h), Color{255, 255, 255});
    }

    void fill(Color color) {
        std::fill(pixels.begin(), pixels.end(), color);
    }

    void set_pixel(int x, int y, Color color) {
        if (x < 0 || y < 0 || x >= width || y >= height) {
            return;
        }
        pixels[static_cast<std::size_t>(y * width + x)] = color;
    }
};

struct RenderResult {
    Image image;
    std::vector<std::string> figures;
    std::vector<std::string> operations;
};

class Renderer {
public:
    RenderResult render(const ProgramPtr& program) {
        vars_ = {{"PI", pi}, {"pi", pi}, {"E", std::exp(1.0)}, {"e", std::exp(1.0)}};
        for (const auto& statement : program->statements) {
            execute(statement);
        }
        if (image_.width == 0 || image_.height == 0) {
            throw std::runtime_error("program did not create a canvas");
        }
        std::vector<std::string> names;
        for (const auto& item : figures_) {
            names.push_back(item.first);
        }
        std::sort(names.begin(), names.end());
        return RenderResult{image_, names, operations_};
    }

private:
    static constexpr double pi = 3.14159265358979323846;
    Image image_;
    std::unordered_map<std::string, double> vars_;
    std::unordered_map<std::string, FigureBlockPtr> figures_;
    std::vector<std::string> operations_;
    Color stroke_{0x11, 0x18, 0x27};
    Color fill_{0xff, 0xff, 0xff};
    bool fill_on_ = false;
    double line_width_ = 1.0;

    double eval(const ExprPtr& expr) {
        switch (expr->kind) {
        case Expr::Kind::Number:
            return expr->number;
        case Expr::Kind::Variable: {
            const auto it = vars_.find(expr->name);
            if (it == vars_.end()) {
                throw std::runtime_error("undefined variable " + expr->name);
            }
            return it->second;
        }
        case Expr::Kind::Unary:
            if (expr->op == '-') {
                return -eval(expr->arg);
            }
            break;
        case Expr::Kind::Binary: {
            const double left = eval(expr->left);
            const double right = eval(expr->right);
            if (expr->op == '+') {
                return left + right;
            }
            if (expr->op == '-') {
                return left - right;
            }
            if (expr->op == '*') {
                return left * right;
            }
            if (expr->op == '/') {
                if (right == 0.0) {
                    throw std::runtime_error("division by zero");
                }
                return left / right;
            }
            break;
        }
        case Expr::Kind::Call:
            return call(expr->name, eval(expr->arg));
        }
        throw std::runtime_error("unsupported expression");
    }

    static double call(const std::string& name, double arg) {
        if (name == "sin") return std::sin(arg);
        if (name == "cos") return std::cos(arg);
        if (name == "tan") return std::tan(arg);
        if (name == "ln") return std::log(arg);
        if (name == "sqrt") return std::sqrt(arg);
        if (name == "sqr") return arg * arg;
        if (name == "exp") return std::exp(arg);
        throw std::runtime_error("unsupported function " + name);
    }

    static int round_dimension(double value) {
        return static_cast<int>(std::floor(value + 0.5));
    }

    void fill_circle(double cx, double cy, double radius, Color color) {
        radius = std::abs(radius);
        const double rr = radius * radius;
        for (int y = static_cast<int>(std::floor(cy - radius)); y <= static_cast<int>(std::ceil(cy + radius)); ++y) {
            for (int x = static_cast<int>(std::floor(cx - radius)); x <= static_cast<int>(std::ceil(cx + radius)); ++x) {
                const double dx = static_cast<double>(x) - cx;
                const double dy = static_cast<double>(y) - cy;
                if (dx * dx + dy * dy <= rr) {
                    image_.set_pixel(x, y, color);
                }
            }
        }
    }

    void draw_line(double x1, double y1, double x2, double y2, Color color, double width) {
        const double dx = x2 - x1;
        const double dy = y2 - y1;
        const int steps = static_cast<int>(std::max(std::abs(dx), std::abs(dy)));
        if (steps == 0) {
            fill_circle(x1, y1, std::max(1.0, width / 2.0), color);
            return;
        }
        const double radius = std::max(0.5, width / 2.0);
        for (int i = 0; i <= steps; ++i) {
            const double t = static_cast<double>(i) / static_cast<double>(steps);
            fill_circle(x1 + dx * t, y1 + dy * t, radius, color);
        }
    }

    void draw_box(double x1, double y1, double x2, double y2) {
        const double left = std::min(x1, x2);
        const double right = std::max(x1, x2);
        const double top = std::min(y1, y2);
        const double bottom = std::max(y1, y2);
        if (fill_on_) {
            for (int y = round_dimension(top); y <= round_dimension(bottom); ++y) {
                for (int x = round_dimension(left); x <= round_dimension(right); ++x) {
                    image_.set_pixel(x, y, fill_);
                }
            }
        }
        draw_line(left, top, right, top, stroke_, line_width_);
        draw_line(right, top, right, bottom, stroke_, line_width_);
        draw_line(right, bottom, left, bottom, stroke_, line_width_);
        draw_line(left, bottom, left, top, stroke_, line_width_);
    }

    void draw_circle(double cx, double cy, double radius) {
        radius = std::abs(radius);
        if (fill_on_) {
            fill_circle(cx, cy, radius, fill_);
        }
        const int steps = static_cast<int>(std::max(24.0, radius * 8.0));
        double prev_x = 0.0;
        double prev_y = 0.0;
        for (int i = 0; i <= steps; ++i) {
            const double angle = 2.0 * pi * static_cast<double>(i) / static_cast<double>(steps);
            const double x = cx + std::cos(angle) * radius;
            const double y = cy + std::sin(angle) * radius;
            if (i > 0) {
                draw_line(prev_x, prev_y, x, y, stroke_, std::max(1.0, line_width_));
            }
            prev_x = x;
            prev_y = y;
        }
    }

    void execute_figure(const FigureBlockPtr& block) {
        for (const auto& statement : block->statements) {
            execute(statement);
        }
    }

    void execute_figure_ref(const FigureRefPtr& ref) {
        if (!ref->named) {
            execute_figure(ref->block);
            return;
        }
        const auto it = figures_.find(ref->name);
        if (it == figures_.end()) {
            throw std::runtime_error("undefined figure " + ref->name);
        }
        execute_figure(it->second);
    }

    void execute_primitive(const StatementPtr& statement) {
        std::vector<double> args;
        args.reserve(statement->exprs.size());
        for (const auto& expr : statement->exprs) {
            args.push_back(eval(expr));
        }
        if (statement->primitive == "point") {
            fill_circle(args[0], args[1], std::max(1.0, line_width_ / 2.0), stroke_);
        } else if (statement->primitive == "line") {
            draw_line(args[0], args[1], args[2], args[3], stroke_, line_width_);
        } else if (statement->primitive == "box") {
            draw_box(args[0], args[1], args[2], args[3]);
        } else if (statement->primitive == "circle") {
            draw_circle(args[0], args[1], args[2]);
        } else {
            throw std::runtime_error("unsupported primitive " + statement->primitive);
        }
        operations_.push_back(statement->primitive + " " + std::to_string(args.size()) + " args");
    }

    void execute(const StatementPtr& statement) {
        switch (statement->kind) {
        case Statement::Kind::Canvas: {
            const int width = round_dimension(eval(statement->exprs[0]));
            const int height = round_dimension(eval(statement->exprs[1]));
            image_.reset(width, height);
            image_.fill(Color{255, 255, 255});
            operations_.push_back("canvas " + std::to_string(width) + "," + std::to_string(height));
            break;
        }
        case Statement::Kind::Background:
            image_.fill(statement->color);
            operations_.push_back("background " + color_hex(statement->color));
            break;
        case Statement::Kind::Stroke:
            stroke_ = statement->color;
            break;
        case Statement::Kind::Fill:
            fill_ = statement->color;
            fill_on_ = statement->enabled;
            break;
        case Statement::Kind::Width:
            line_width_ = std::max(1.0, eval(statement->exprs[0]));
            break;
        case Statement::Kind::Assign:
            vars_[statement->name] = eval(statement->exprs[0]);
            break;
        case Statement::Kind::DefineFigure:
            figures_[statement->name] = statement->figure;
            operations_.push_back("define " + statement->name);
            break;
        case Statement::Kind::Draw:
            execute_figure_ref(statement->target);
            break;
        case Statement::Kind::Repdraw: {
            const int count = round_dimension(eval(statement->exprs[0]));
            if (count < 0 || count > max_repdraw_iterations) {
                throw std::runtime_error("repdraw count is outside supported range");
            }
            for (int i = 0; i < count; ++i) {
                execute_figure_ref(statement->target);
            }
            break;
        }
        case Statement::Kind::Primitive:
            execute_primitive(statement);
            break;
        }
    }
};

static ProgramPtr parse_program(const std::string& source) {
    return std::any_cast<ProgramPtr>(draw::parse_value(draw::tokenize(source), make_reducers()));
}

static void write_u32_be(std::ostream& output, std::uint32_t value) {
    output.put(static_cast<char>((value >> 24) & 0xff));
    output.put(static_cast<char>((value >> 16) & 0xff));
    output.put(static_cast<char>((value >> 8) & 0xff));
    output.put(static_cast<char>(value & 0xff));
}

static void append_u32_be(std::vector<std::uint8_t>& out, std::uint32_t value) {
    out.push_back(static_cast<std::uint8_t>((value >> 24) & 0xff));
    out.push_back(static_cast<std::uint8_t>((value >> 16) & 0xff));
    out.push_back(static_cast<std::uint8_t>((value >> 8) & 0xff));
    out.push_back(static_cast<std::uint8_t>(value & 0xff));
}

static std::uint32_t update_crc(std::uint32_t crc, std::uint8_t byte) {
    crc ^= byte;
    for (int i = 0; i < 8; ++i) {
        crc = (crc & 1U) != 0 ? 0xedb88320U ^ (crc >> 1U) : crc >> 1U;
    }
    return crc;
}

static std::uint32_t crc32(const char type[4], const std::vector<std::uint8_t>& data) {
    std::uint32_t crc = 0xffffffffU;
    for (int i = 0; i < 4; ++i) {
        crc = update_crc(crc, static_cast<std::uint8_t>(type[i]));
    }
    for (const auto byte : data) {
        crc = update_crc(crc, byte);
    }
    return crc ^ 0xffffffffU;
}

static std::uint32_t adler32(const std::vector<std::uint8_t>& data) {
    constexpr std::uint32_t mod = 65521;
    std::uint32_t a = 1;
    std::uint32_t b = 0;
    for (const auto byte : data) {
        a = (a + byte) % mod;
        b = (b + a) % mod;
    }
    return (b << 16U) | a;
}

static std::vector<std::uint8_t> png_scanlines(const Image& image) {
    std::vector<std::uint8_t> raw;
    raw.reserve(static_cast<std::size_t>(image.height) * (static_cast<std::size_t>(image.width) * 3 + 1));
    for (int y = 0; y < image.height; ++y) {
        raw.push_back(0); // PNG filter type 0: none.
        for (int x = 0; x < image.width; ++x) {
            const auto& pixel = image.pixels[static_cast<std::size_t>(y * image.width + x)];
            raw.push_back(pixel.r);
            raw.push_back(pixel.g);
            raw.push_back(pixel.b);
        }
    }
    return raw;
}

static std::vector<std::uint8_t> zlib_store(const std::vector<std::uint8_t>& raw) {
    std::vector<std::uint8_t> out;
    out.reserve(raw.size() + raw.size() / 65535 * 5 + 16);
    out.push_back(0x78); // zlib header: deflate, 32K window.
    out.push_back(0x01); // fastest/no compression check bits.
    std::size_t offset = 0;
    while (offset < raw.size()) {
        const std::size_t remaining = raw.size() - offset;
        const std::uint16_t block_size = static_cast<std::uint16_t>(std::min<std::size_t>(remaining, 65535));
        const bool final = offset + block_size == raw.size();
        out.push_back(final ? 0x01 : 0x00);
        out.push_back(static_cast<std::uint8_t>(block_size & 0xff));
        out.push_back(static_cast<std::uint8_t>((block_size >> 8) & 0xff));
        const std::uint16_t nlen = static_cast<std::uint16_t>(~block_size);
        out.push_back(static_cast<std::uint8_t>(nlen & 0xff));
        out.push_back(static_cast<std::uint8_t>((nlen >> 8) & 0xff));
        out.insert(out.end(), raw.begin() + static_cast<std::ptrdiff_t>(offset), raw.begin() + static_cast<std::ptrdiff_t>(offset + block_size));
        offset += block_size;
    }
    append_u32_be(out, adler32(raw));
    return out;
}

static void write_chunk(std::ostream& output, const char type[4], const std::vector<std::uint8_t>& data) {
    write_u32_be(output, static_cast<std::uint32_t>(data.size()));
    output.write(type, 4);
    if (!data.empty()) {
        output.write(reinterpret_cast<const char*>(data.data()), static_cast<std::streamsize>(data.size()));
    }
    write_u32_be(output, crc32(type, data));
}

static void write_png(const std::string& path, const Image& image) {
    ensure_parent_dir(path);
    std::ofstream output(path, std::ios::binary);
    if (!output) {
        throw std::runtime_error("cannot open output image: " + path);
    }
    const std::array<std::uint8_t, 8> signature{{137, 80, 78, 71, 13, 10, 26, 10}};
    output.write(reinterpret_cast<const char*>(signature.data()), static_cast<std::streamsize>(signature.size()));

    std::vector<std::uint8_t> ihdr;
    ihdr.reserve(13);
    append_u32_be(ihdr, static_cast<std::uint32_t>(image.width));
    append_u32_be(ihdr, static_cast<std::uint32_t>(image.height));
    ihdr.push_back(8);  // 8-bit channel depth.
    ihdr.push_back(2);  // truecolor RGB.
    ihdr.push_back(0);  // deflate compression.
    ihdr.push_back(0);  // adaptive filtering.
    ihdr.push_back(0);  // no interlace.
    write_chunk(output, "IHDR", ihdr);
    write_chunk(output, "IDAT", zlib_store(png_scanlines(image)));
    write_chunk(output, "IEND", {});
}

static RenderResult render_source(const std::string& source, const std::string& output_path) {
    Renderer renderer;
    RenderResult result = renderer.render(parse_program(source));
    write_png(output_path, result.image);
    return result;
}

static std::string build_report(const std::string& input_path, const std::string& output_path, const RenderResult& result) {
    std::map<std::string, int> counts;
    for (const auto& op : result.operations) {
        ++counts[op];
    }
    std::ostringstream report;
    report << "DRAW C++ render report\n";
    report << "Source: " << input_path << "\n";
    report << "Output: " << output_path << "\n";
    report << "Canvas: " << result.image.width << "x" << result.image.height << "\n";
    report << "Figures: [";
    for (std::size_t i = 0; i < result.figures.size(); ++i) {
        report << (i == 0 ? "" : ", ") << result.figures[i];
    }
    report << "]\n\nOperation summary:\n";
    for (const auto& item : counts) {
        report << "  " << item.first << ": " << item.second << "\n";
    }
    return report.str();
}

static void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

static int count_operation(const RenderResult& result, const std::string& name) {
    return static_cast<int>(std::count(result.operations.begin(), result.operations.end(), name));
}

static bool has_png_signature(const std::string& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input) {
        return false;
    }
    std::array<unsigned char, 8> header{};
    input.read(reinterpret_cast<char*>(header.data()), static_cast<std::streamsize>(header.size()));
    return input.gcount() == static_cast<std::streamsize>(header.size()) &&
           header == std::array<unsigned char, 8>{{137, 80, 78, 71, 13, 10, 26, 10}};
}

static void run_assertions(const std::string& source, const std::string& output_path) {
    const RenderResult result = render_source(source, output_path);
    require(result.image.width == 960 && result.image.height == 640, "expected 960x640 canvas");
    require(has_png_signature(output_path), "expected PNG output");
    require(count_operation(result, "line 4 args") == 90, "expected 90 rendered lines");
    require(count_operation(result, "circle 3 args") == 196, "expected 196 rendered circles");
    require(count_operation(result, "box 4 args") == 2, "expected 2 rendered boxes");

    draw::Parser parser(make_reducers());
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
        const std::string output_path = read_option(args, "--output", "dist/sample-cpp.png");
        const std::string log_path = read_option(args, "--log", "dist/draw-cpp-demo.log");
        const std::string input_path = args.empty() ? "sample.draw" : args.front();
        const std::string source = read_text_file(input_path);

        if (assert_mode) {
            run_assertions(source, output_path);
        }

        const RenderResult result = render_source(source, output_path);
        const std::string report = build_report(input_path, output_path, result);
        write_text_file(log_path, report);
        std::cout << report;
        return 0;
    } catch (const std::exception& ex) {
        std::cerr << ex.what() << "\n";
        return 1;
    }
}
