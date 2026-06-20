#include "renderer.hpp"

#include <algorithm>
#include <cmath>
#include <iomanip>
#include <sstream>
#include <stdexcept>

namespace lfdraw {

constexpr int max_repdraw_iterations = 20000;

void Image::reset(int w, int h) {
    if (w <= 0 || h <= 0 || w > 4096 || h > 4096) {
        throw std::runtime_error("canvas dimensions must be in 1..4096");
    }
    width = w;
    height = h;
    pixels.assign(static_cast<std::size_t>(w * h), Color{255, 255, 255});
}

void Image::fill(Color color) {
    std::fill(pixels.begin(), pixels.end(), color);
}

void Image::set_pixel(int x, int y, Color color) {
    if (x < 0 || y < 0 || x >= width || y >= height) {
        return;
    }
    pixels[static_cast<std::size_t>(y * width + x)] = color;
}

std::string color_hex(Color color) {
    std::ostringstream out;
    out << "#" << std::uppercase << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(color.r)
        << std::setw(2) << static_cast<int>(color.g)
        << std::setw(2) << static_cast<int>(color.b);
    return out.str();
}

RenderResult Renderer::render(const ProgramPtr& program) {
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

double Renderer::eval(const ExprPtr& expr) {
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

double Renderer::call(const std::string& name, double arg) {
    if (name == "sin") return std::sin(arg);
    if (name == "cos") return std::cos(arg);
    if (name == "tan") return std::tan(arg);
    if (name == "ln") return std::log(arg);
    if (name == "sqrt") return std::sqrt(arg);
    if (name == "sqr") return arg * arg;
    if (name == "exp") return std::exp(arg);
    throw std::runtime_error("unsupported function " + name);
}

int Renderer::round_dimension(double value) {
    return static_cast<int>(std::floor(value + 0.5));
}

void Renderer::fill_circle(double cx, double cy, double radius, Color color) {
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

void Renderer::draw_line(double x1, double y1, double x2, double y2, Color color, double width) {
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

void Renderer::draw_box(double x1, double y1, double x2, double y2) {
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

void Renderer::draw_circle(double cx, double cy, double radius) {
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

void Renderer::execute_figure(const FigureBlockPtr& block) {
    for (const auto& statement : block->statements) {
        execute(statement);
    }
}

void Renderer::execute_figure_ref(const FigureRefPtr& ref) {
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

void Renderer::execute_primitive(const StatementPtr& statement) {
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

void Renderer::execute(const StatementPtr& statement) {
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

} // namespace lfdraw
