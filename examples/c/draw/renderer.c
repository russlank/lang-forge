#include "renderer.h"

#include <math.h>
#include <stdlib.h>
#include <string.h>

void draw_renderer_init(draw_renderer *renderer) {
    memset(renderer, 0, sizeof(*renderer));
    renderer->stroke.r = 0x11;
    renderer->stroke.g = 0x18;
    renderer->stroke.b = 0x27;
    renderer->fill.r = 0xff;
    renderer->fill.g = 0xff;
    renderer->fill.b = 0xff;
    renderer->line_width = 1.0;
}

static int draw_set_var(draw_renderer *renderer, const char *name, double value, char *message, size_t message_size) {
    draw_var *item = renderer->vars;
    while (item != NULL) {
        if (strcmp(item->name, name) == 0) {
            item->value = value;
            return 1;
        }
        item = item->next;
    }
    item = (draw_var *)calloc(1, sizeof(draw_var));
    if (item == NULL) {
        return demo_set_error(message, message_size, "out of memory storing variable");
    }
    item->name = (char *)name;
    item->value = value;
    item->next = renderer->vars;
    renderer->vars = item;
    return 1;
}

static int draw_get_var(draw_renderer *renderer, const char *name, double *out) {
    draw_var *item = renderer->vars;
    while (item != NULL) {
        if (strcmp(item->name, name) == 0) {
            *out = item->value;
            return 1;
        }
        item = item->next;
    }
    return 0;
}

size_t draw_var_count(draw_renderer *renderer) {
    size_t count = 0;
    draw_var *item = renderer->vars;
    while (item != NULL) {
        count++;
        item = item->next;
    }
    return count;
}

static int draw_set_figure(draw_renderer *renderer, const char *name, draw_figure_block *block, char *message, size_t message_size) {
    draw_named_figure *item = renderer->figures;
    while (item != NULL) {
        if (strcmp(item->name, name) == 0) {
            item->block = block;
            return 1;
        }
        item = item->next;
    }
    item = (draw_named_figure *)calloc(1, sizeof(draw_named_figure));
    if (item == NULL) {
        return demo_set_error(message, message_size, "out of memory storing figure");
    }
    item->name = (char *)name;
    item->block = block;
    item->next = renderer->figures;
    renderer->figures = item;
    return 1;
}

static draw_figure_block *draw_get_figure(draw_renderer *renderer, const char *name) {
    draw_named_figure *item = renderer->figures;
    while (item != NULL) {
        if (strcmp(item->name, name) == 0) {
            return item->block;
        }
        item = item->next;
    }
    return NULL;
}

size_t draw_figure_count(draw_renderer *renderer) {
    size_t count = 0;
    draw_named_figure *item = renderer->figures;
    while (item != NULL) {
        count++;
        item = item->next;
    }
    return count;
}

void draw_renderer_free(draw_renderer *renderer) {
    draw_var *var = renderer->vars;
    draw_named_figure *figure = renderer->figures;
    while (var != NULL) {
        draw_var *next = var->next;
        free(var);
        var = next;
    }
    while (figure != NULL) {
        draw_named_figure *next = figure->next;
        free(figure);
        figure = next;
    }
    demo_image_free(&renderer->image);
}

static int draw_eval(draw_renderer *renderer, draw_expr *expr, double *out, char *message, size_t message_size);

static int draw_eval_call(draw_renderer *renderer, draw_expr *expr, double *out, char *message, size_t message_size) {
    double arg = 0.0;
    if (!draw_eval(renderer, expr->arg, &arg, message, message_size)) {
        return 0;
    }
    if (strcmp(expr->name, "sin") == 0) {
        *out = sin(arg);
    } else if (strcmp(expr->name, "cos") == 0) {
        *out = cos(arg);
    } else if (strcmp(expr->name, "tan") == 0) {
        *out = tan(arg);
    } else if (strcmp(expr->name, "ln") == 0) {
        *out = log(arg);
    } else if (strcmp(expr->name, "sqrt") == 0) {
        *out = sqrt(arg);
    } else if (strcmp(expr->name, "sqr") == 0) {
        *out = arg * arg;
    } else if (strcmp(expr->name, "exp") == 0) {
        *out = exp(arg);
    } else {
        return demo_set_error(message, message_size, "unsupported function %s", expr->name);
    }
    return 1;
}

static int draw_eval(draw_renderer *renderer, draw_expr *expr, double *out, char *message, size_t message_size) {
    double left = 0.0;
    double right = 0.0;
    switch (expr->kind) {
    case DRAW_EXPR_NUMBER:
        *out = expr->number;
        return 1;
    case DRAW_EXPR_VARIABLE:
        if (!draw_get_var(renderer, expr->name, out)) {
            return demo_set_error(message, message_size, "undefined variable %s", expr->name);
        }
        return 1;
    case DRAW_EXPR_UNARY:
        if (!draw_eval(renderer, expr->arg, out, message, message_size)) {
            return 0;
        }
        if (expr->op == '-') {
            *out = -*out;
            return 1;
        }
        return demo_set_error(message, message_size, "unsupported unary operator %c", expr->op);
    case DRAW_EXPR_BINARY:
        if (!draw_eval(renderer, expr->left, &left, message, message_size) ||
            !draw_eval(renderer, expr->right, &right, message, message_size)) {
            return 0;
        }
        if (expr->op == '+') {
            *out = left + right;
        } else if (expr->op == '-') {
            *out = left - right;
        } else if (expr->op == '*') {
            *out = left * right;
        } else if (expr->op == '/') {
            if (right == 0.0) {
                return demo_set_error(message, message_size, "division by zero");
            }
            *out = left / right;
        } else {
            return demo_set_error(message, message_size, "unsupported binary operator %c", expr->op);
        }
        return 1;
    case DRAW_EXPR_CALL:
        return draw_eval_call(renderer, expr, out, message, message_size);
    }
    return demo_set_error(message, message_size, "unsupported expression");
}

static int draw_round_dimension(double value) {
    return (int)floor(value + 0.5);
}

static void draw_fill_circle(draw_renderer *renderer, double cx, double cy, double radius, draw_color color) {
    int x = 0;
    int y = 0;
    int min_x = 0;
    int max_x = 0;
    int min_y = 0;
    int max_y = 0;
    double rr = 0.0;
    if (radius < 0) {
        radius = -radius;
    }
    rr = radius * radius;
    min_x = (int)floor(cx - radius);
    max_x = (int)ceil(cx + radius);
    min_y = (int)floor(cy - radius);
    max_y = (int)ceil(cy + radius);
    for (y = min_y; y <= max_y; y++) {
        for (x = min_x; x <= max_x; x++) {
            double dx = (double)x - cx;
            double dy = (double)y - cy;
            if (dx * dx + dy * dy <= rr) {
                demo_image_set_pixel(&renderer->image, x, y, color.r, color.g, color.b);
            }
        }
    }
}

static void draw_line(draw_renderer *renderer, double x1, double y1, double x2, double y2, draw_color color, double width) {
    double dx = x2 - x1;
    double dy = y2 - y1;
    int steps = (int)fmax(fabs(dx), fabs(dy));
    int i = 0;
    double radius = fmax(0.5, width / 2.0);
    if (steps == 0) {
        draw_fill_circle(renderer, x1, y1, fmax(1.0, width / 2.0), color);
        return;
    }
    for (i = 0; i <= steps; i++) {
        double t = (double)i / (double)steps;
        draw_fill_circle(renderer, x1 + dx * t, y1 + dy * t, radius, color);
    }
}

static void draw_box(draw_renderer *renderer, double x1, double y1, double x2, double y2) {
    double left = fmin(x1, x2);
    double right = fmax(x1, x2);
    double top = fmin(y1, y2);
    double bottom = fmax(y1, y2);
    int x = 0;
    int y = 0;
    if (renderer->fill_on) {
        for (y = draw_round_dimension(top); y <= draw_round_dimension(bottom); y++) {
            for (x = draw_round_dimension(left); x <= draw_round_dimension(right); x++) {
                demo_image_set_pixel(&renderer->image, x, y, renderer->fill.r, renderer->fill.g, renderer->fill.b);
            }
        }
    }
    draw_line(renderer, left, top, right, top, renderer->stroke, renderer->line_width);
    draw_line(renderer, right, top, right, bottom, renderer->stroke, renderer->line_width);
    draw_line(renderer, right, bottom, left, bottom, renderer->stroke, renderer->line_width);
    draw_line(renderer, left, bottom, left, top, renderer->stroke, renderer->line_width);
}

static void draw_circle(draw_renderer *renderer, double cx, double cy, double radius) {
    int steps = 0;
    int i = 0;
    double prev_x = 0.0;
    double prev_y = 0.0;
    if (radius < 0) {
        radius = -radius;
    }
    if (renderer->fill_on) {
        draw_fill_circle(renderer, cx, cy, radius, renderer->fill);
    }
    steps = (int)fmax(24.0, radius * 8.0);
    for (i = 0; i <= steps; i++) {
        double angle = 2.0 * 3.14159265358979323846 * (double)i / (double)steps;
        double x = cx + cos(angle) * radius;
        double y = cy + sin(angle) * radius;
        if (i > 0) {
            draw_line(renderer, prev_x, prev_y, x, y, renderer->stroke, fmax(1.0, renderer->line_width));
        }
        prev_x = x;
        prev_y = y;
    }
}

static int draw_execute_statement(draw_renderer *renderer, draw_statement *statement, char *message, size_t message_size);

static int draw_execute_figure(draw_renderer *renderer, draw_figure_block *block, char *message, size_t message_size) {
    draw_statement_node *node = block == NULL || block->statements == NULL ? NULL : block->statements->head;
    while (node != NULL) {
        if (!draw_execute_statement(renderer, node->statement, message, message_size)) {
            return 0;
        }
        node = node->next;
    }
    return 1;
}

static int draw_execute_figure_ref(draw_renderer *renderer, draw_figure_ref *ref, char *message, size_t message_size) {
    if (ref->kind == DRAW_FIGURE_INLINE) {
        return draw_execute_figure(renderer, ref->block, message, message_size);
    }
    if (ref->kind == DRAW_FIGURE_NAMED) {
        draw_figure_block *block = draw_get_figure(renderer, ref->name);
        if (block == NULL) {
            return demo_set_error(message, message_size, "undefined figure %s", ref->name);
        }
        return draw_execute_figure(renderer, block, message, message_size);
    }
    return demo_set_error(message, message_size, "unsupported figure reference");
}

static int draw_execute_primitive(draw_renderer *renderer, draw_statement *statement, char *message, size_t message_size) {
    double args[4] = {0.0, 0.0, 0.0, 0.0};
    size_t i = 0;
    if (!renderer->has_image) {
        return demo_set_error(message, message_size, "drawing command used before canvas");
    }
    for (i = 0; i < statement->expr_count; i++) {
        if (!draw_eval(renderer, statement->exprs[i], &args[i], message, message_size)) {
            return 0;
        }
    }
    if (strcmp(statement->primitive, "point") == 0) {
        draw_fill_circle(renderer, args[0], args[1], fmax(1.0, renderer->line_width / 2.0), renderer->stroke);
        renderer->counts.point++;
    } else if (strcmp(statement->primitive, "line") == 0) {
        draw_line(renderer, args[0], args[1], args[2], args[3], renderer->stroke, renderer->line_width);
        renderer->counts.line++;
    } else if (strcmp(statement->primitive, "box") == 0) {
        draw_box(renderer, args[0], args[1], args[2], args[3]);
        renderer->counts.box++;
    } else if (strcmp(statement->primitive, "circle") == 0) {
        draw_circle(renderer, args[0], args[1], args[2]);
        renderer->counts.circle++;
    } else {
        return demo_set_error(message, message_size, "unsupported primitive %s", statement->primitive);
    }
    return 1;
}

static int draw_execute_statement(draw_renderer *renderer, draw_statement *statement, char *message, size_t message_size) {
    double first = 0.0;
    double second = 0.0;
    switch (statement->kind) {
    case DRAW_STMT_CANVAS:
        if (!draw_eval(renderer, statement->exprs[0], &first, message, message_size) ||
            !draw_eval(renderer, statement->exprs[1], &second, message, message_size)) {
            return 0;
        }
        renderer->image.width = 0;
        renderer->image.height = 0;
        renderer->image.rgb = NULL;
        if (!demo_image_init(&renderer->image, draw_round_dimension(first), draw_round_dimension(second), message, message_size)) {
            return 0;
        }
        renderer->has_image = 1;
        renderer->counts.canvas++;
        demo_image_fill(&renderer->image, 255, 255, 255);
        return 1;
    case DRAW_STMT_BACKGROUND:
        if (!renderer->has_image) {
            return demo_set_error(message, message_size, "background used before canvas");
        }
        renderer->background = statement->color;
        renderer->counts.background++;
        demo_image_fill(&renderer->image, statement->color.r, statement->color.g, statement->color.b);
        return 1;
    case DRAW_STMT_STROKE:
        renderer->stroke = statement->color;
        return 1;
    case DRAW_STMT_FILL:
        renderer->fill = statement->color;
        renderer->fill_on = statement->enabled;
        return 1;
    case DRAW_STMT_WIDTH:
        if (!draw_eval(renderer, statement->exprs[0], &first, message, message_size)) {
            return 0;
        }
        renderer->line_width = fmax(1.0, first);
        return 1;
    case DRAW_STMT_ASSIGN:
        if (!draw_eval(renderer, statement->exprs[0], &first, message, message_size)) {
            return 0;
        }
        return draw_set_var(renderer, statement->name, first, message, message_size);
    case DRAW_STMT_DEFINE_FIGURE:
        renderer->counts.define++;
        return draw_set_figure(renderer, statement->name, statement->figure, message, message_size);
    case DRAW_STMT_DRAW:
        return draw_execute_figure_ref(renderer, statement->target, message, message_size);
    case DRAW_STMT_REPDRAW: {
        int i = 0;
        int count = 0;
        if (!draw_eval(renderer, statement->exprs[0], &first, message, message_size)) {
            return 0;
        }
        count = draw_round_dimension(first);
        if (count < 0 || count > DRAW_MAX_REPDRAW_ITERATIONS) {
            return demo_set_error(message, message_size, "repdraw count %d is outside 0..%d", count, DRAW_MAX_REPDRAW_ITERATIONS);
        }
        for (i = 0; i < count; i++) {
            if (!draw_execute_figure_ref(renderer, statement->target, message, message_size)) {
                return 0;
            }
        }
        return 1;
    }
    case DRAW_STMT_PRIMITIVE:
        return draw_execute_primitive(renderer, statement, message, message_size);
    }
    return demo_set_error(message, message_size, "unsupported statement");
}

int draw_render(draw_program *program, draw_renderer *renderer, char *message, size_t message_size) {
    draw_statement_node *node = program == NULL || program->statements == NULL ? NULL : program->statements->head;
    draw_renderer_init(renderer);
    if (!draw_set_var(renderer, "PI", 3.14159265358979323846, message, message_size) ||
        !draw_set_var(renderer, "pi", 3.14159265358979323846, message, message_size) ||
        !draw_set_var(renderer, "E", 2.71828182845904523536, message, message_size) ||
        !draw_set_var(renderer, "e", 2.71828182845904523536, message, message_size)) {
        return 0;
    }
    while (node != NULL) {
        if (!draw_execute_statement(renderer, node->statement, message, message_size)) {
            return 0;
        }
        node = node->next;
    }
    if (!renderer->has_image) {
        return demo_set_error(message, message_size, "program did not create a canvas");
    }
    return 1;
}
