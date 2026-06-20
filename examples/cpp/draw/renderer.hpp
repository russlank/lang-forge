#pragma once

#include "ast.hpp"

#include <string>
#include <unordered_map>
#include <vector>

namespace lfdraw {

/// In-memory RGB image.
struct Image {
    int width = 0;
    int height = 0;
    std::vector<Color> pixels;

    void reset(int w, int h);
    void fill(Color color);
    void set_pixel(int x, int y, Color color);
};

/// Rendering result plus metadata used by assertions and reports.
struct RenderResult {
    Image image;
    std::vector<std::string> figures;
    std::vector<std::string> operations;
};

/// Formats an RGB color as a CSS-style hex string.
std::string color_hex(Color color);

/// Executes a DRAW AST and paints it into an in-memory image.
class Renderer {
public:
    RenderResult render(const ProgramPtr& program);

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

    double eval(const ExprPtr& expr);
    static double call(const std::string& name, double arg);
    static int round_dimension(double value);
    void fill_circle(double cx, double cy, double radius, Color color);
    void draw_line(double x1, double y1, double x2, double y2, Color color, double width);
    void draw_box(double x1, double y1, double x2, double y2);
    void draw_circle(double cx, double cy, double radius);
    void execute_figure(const FigureBlockPtr& block);
    void execute_figure_ref(const FigureRefPtr& ref);
    void execute_primitive(const StatementPtr& statement);
    void execute(const StatementPtr& statement);
};

} // namespace lfdraw
