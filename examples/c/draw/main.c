#include "demo.h"
#include "parser.h"

#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define DRAW_MAX_REPDRAW_ITERATIONS 20000

/* The generated parser supplies reduction events; this file owns the semantic
 * model that those reductions build. Keeping the AST handwritten makes the
 * generated C API small while still showing an end-to-end compiler pipeline. */
typedef struct draw_color {
    unsigned char r;
    unsigned char g;
    unsigned char b;
} draw_color;

typedef enum draw_expr_kind {
    DRAW_EXPR_NUMBER,
    DRAW_EXPR_VARIABLE,
    DRAW_EXPR_UNARY,
    DRAW_EXPR_BINARY,
    DRAW_EXPR_CALL
} draw_expr_kind;

typedef struct draw_expr {
    draw_expr_kind kind;
    double number;
    char *name;
    char op;
    struct draw_expr *left;
    struct draw_expr *right;
    struct draw_expr *arg;
} draw_expr;

typedef enum draw_statement_kind {
    DRAW_STMT_CANVAS,
    DRAW_STMT_BACKGROUND,
    DRAW_STMT_STROKE,
    DRAW_STMT_FILL,
    DRAW_STMT_WIDTH,
    DRAW_STMT_ASSIGN,
    DRAW_STMT_DEFINE_FIGURE,
    DRAW_STMT_DRAW,
    DRAW_STMT_REPDRAW,
    DRAW_STMT_PRIMITIVE
} draw_statement_kind;

typedef struct draw_statement draw_statement;
typedef struct draw_statement_node draw_statement_node;

typedef struct draw_statement_list {
    draw_statement_node *head;
    draw_statement_node *tail;
    size_t count;
} draw_statement_list;

struct draw_statement_node {
    draw_statement *statement;
    draw_statement_node *next;
};

typedef struct draw_figure_block {
    draw_statement_list *statements;
} draw_figure_block;

typedef enum draw_figure_ref_kind {
    DRAW_FIGURE_NAMED,
    DRAW_FIGURE_INLINE
} draw_figure_ref_kind;

typedef struct draw_figure_ref {
    draw_figure_ref_kind kind;
    char *name;
    draw_figure_block *block;
} draw_figure_ref;

struct draw_statement {
    draw_statement_kind kind;
    char *name;
    char primitive[8];
    draw_color color;
    int enabled;
    draw_expr *exprs[4];
    size_t expr_count;
    draw_figure_block *figure;
    draw_figure_ref *target;
};

typedef struct draw_program {
    draw_statement_list *statements;
} draw_program;

typedef struct draw_binary_tail {
    char op;
    draw_expr *right;
    struct draw_binary_tail *next;
} draw_binary_tail;

typedef struct draw_binary_tail_list {
    draw_binary_tail *head;
    draw_binary_tail *tail;
    size_t count;
} draw_binary_tail_list;

typedef struct draw_var {
    char *name;
    double value;
    struct draw_var *next;
} draw_var;

typedef struct draw_named_figure {
    char *name;
    draw_figure_block *block;
    struct draw_named_figure *next;
} draw_named_figure;

typedef struct draw_counts {
    int canvas;
    int background;
    int point;
    int line;
    int box;
    int circle;
    int define;
} draw_counts;

typedef struct draw_renderer {
    demo_image image;
    int has_image;
    draw_var *vars;
    draw_named_figure *figures;
    draw_color stroke;
    draw_color fill;
    draw_color background;
    int fill_on;
    double line_width;
    draw_counts counts;
} draw_renderer;

typedef struct draw_context {
    demo_arena arena;
} draw_context;

static void *draw_alloc(draw_context *ctx, size_t size, draw_error *error, const char *what) {
    void *ptr = demo_arena_alloc(&ctx->arena, size);
    if (ptr == NULL) {
        snprintf(error->message, sizeof(error->message), "out of memory allocating %s", what);
    }
    return ptr;
}

static char *draw_copy_lexeme(draw_context *ctx, const draw_lexeme *lexeme, draw_error *error) {
    char *copy = demo_arena_copy(&ctx->arena, lexeme->text, lexeme->length);
    if (copy == NULL) {
        snprintf(error->message, sizeof(error->message), "out of memory copying lexeme text");
    }
    return copy;
}

static const draw_lexeme *draw_lexeme_value(draw_value value) {
    return (const draw_lexeme *)value;
}

static draw_expr *draw_expr_number(draw_context *ctx, draw_error *error, double value) {
    draw_expr *expr = (draw_expr *)draw_alloc(ctx, sizeof(draw_expr), error, "number expression");
    if (expr != NULL) {
        expr->kind = DRAW_EXPR_NUMBER;
        expr->number = value;
    }
    return expr;
}

static draw_expr *draw_expr_variable(draw_context *ctx, draw_error *error, char *name) {
    draw_expr *expr = (draw_expr *)draw_alloc(ctx, sizeof(draw_expr), error, "variable expression");
    if (expr != NULL) {
        expr->kind = DRAW_EXPR_VARIABLE;
        expr->name = name;
    }
    return expr;
}

static draw_expr *draw_expr_unary(draw_context *ctx, draw_error *error, char op, draw_expr *arg) {
    draw_expr *expr = (draw_expr *)draw_alloc(ctx, sizeof(draw_expr), error, "unary expression");
    if (expr != NULL) {
        expr->kind = DRAW_EXPR_UNARY;
        expr->op = op;
        expr->arg = arg;
    }
    return expr;
}

static draw_expr *draw_expr_binary(draw_context *ctx, draw_error *error, char op, draw_expr *left, draw_expr *right) {
    draw_expr *expr = (draw_expr *)draw_alloc(ctx, sizeof(draw_expr), error, "binary expression");
    if (expr != NULL) {
        expr->kind = DRAW_EXPR_BINARY;
        expr->op = op;
        expr->left = left;
        expr->right = right;
    }
    return expr;
}

static draw_expr *draw_expr_call(draw_context *ctx, draw_error *error, char *name, draw_expr *arg) {
    draw_expr *expr = (draw_expr *)draw_alloc(ctx, sizeof(draw_expr), error, "call expression");
    if (expr != NULL) {
        expr->kind = DRAW_EXPR_CALL;
        expr->name = name;
        expr->arg = arg;
    }
    return expr;
}

static draw_statement *draw_statement_new(draw_context *ctx, draw_error *error, draw_statement_kind kind) {
    draw_statement *statement = (draw_statement *)draw_alloc(ctx, sizeof(draw_statement), error, "statement");
    if (statement != NULL) {
        statement->kind = kind;
    }
    return statement;
}

static draw_statement_list *draw_statement_list_empty(draw_context *ctx, draw_error *error) {
    return (draw_statement_list *)draw_alloc(ctx, sizeof(draw_statement_list), error, "statement list");
}

static draw_statement_list *draw_statement_list_prepend(draw_context *ctx, draw_error *error, draw_statement *statement, draw_statement_list *tail) {
    draw_statement_node *node = NULL;
    draw_statement_list *list = NULL;
    if (tail == NULL) {
        tail = draw_statement_list_empty(ctx, error);
    }
    if (tail == NULL) {
        return NULL;
    }
    node = (draw_statement_node *)draw_alloc(ctx, sizeof(draw_statement_node), error, "statement list node");
    if (node == NULL) {
        return NULL;
    }
    list = (draw_statement_list *)draw_alloc(ctx, sizeof(draw_statement_list), error, "statement list");
    if (list == NULL) {
        return NULL;
    }
    node->statement = statement;
    node->next = tail->head;
    list->head = node;
    list->tail = tail->tail == NULL ? node : tail->tail;
    list->count = tail->count + 1;
    return list;
}

static draw_binary_tail_list *draw_tail_list_empty(draw_context *ctx, draw_error *error) {
    return (draw_binary_tail_list *)draw_alloc(ctx, sizeof(draw_binary_tail_list), error, "expression tail list");
}

static draw_binary_tail_list *draw_tail_list_prepend(draw_context *ctx, draw_error *error, char op, draw_expr *right, draw_binary_tail_list *tail) {
    draw_binary_tail *node = NULL;
    draw_binary_tail_list *list = NULL;
    if (tail == NULL) {
        tail = draw_tail_list_empty(ctx, error);
    }
    if (tail == NULL) {
        return NULL;
    }
    node = (draw_binary_tail *)draw_alloc(ctx, sizeof(draw_binary_tail), error, "expression tail");
    if (node == NULL) {
        return NULL;
    }
    list = (draw_binary_tail_list *)draw_alloc(ctx, sizeof(draw_binary_tail_list), error, "expression tail list");
    if (list == NULL) {
        return NULL;
    }
    node->op = op;
    node->right = right;
    node->next = tail->head;
    list->head = node;
    list->tail = tail->tail == NULL ? node : tail->tail;
    list->count = tail->count + 1;
    return list;
}

static draw_expr *draw_fold_binary(draw_context *ctx, draw_error *error, draw_expr *left, draw_binary_tail_list *tails) {
    draw_binary_tail *tail = tails == NULL ? NULL : tails->head;
    draw_expr *result = left;
    while (tail != NULL) {
        result = draw_expr_binary(ctx, error, tail->op, result, tail->right);
        if (result == NULL) {
            return NULL;
        }
        tail = tail->next;
    }
    return result;
}

static int draw_hex(char ch) {
    if (ch >= '0' && ch <= '9') {
        return ch - '0';
    }
    if (ch >= 'a' && ch <= 'f') {
        return ch - 'a' + 10;
    }
    if (ch >= 'A' && ch <= 'F') {
        return ch - 'A' + 10;
    }
    return 0;
}

static draw_color *draw_parse_color(draw_context *ctx, draw_error *error, const draw_lexeme *lexeme) {
    draw_color *color = (draw_color *)draw_alloc(ctx, sizeof(draw_color), error, "color");
    if (color == NULL) {
        return NULL;
    }
    color->r = (unsigned char)(draw_hex(lexeme->text[1]) * 16 + draw_hex(lexeme->text[2]));
    color->g = (unsigned char)(draw_hex(lexeme->text[3]) * 16 + draw_hex(lexeme->text[4]));
    color->b = (unsigned char)(draw_hex(lexeme->text[5]) * 16 + draw_hex(lexeme->text[6]));
    return color;
}

static draw_statement *draw_primitive(draw_context *ctx, draw_error *error, const char *kind, size_t count, draw_expr *a, draw_expr *b, draw_expr *c, draw_expr *d) {
    draw_statement *statement = draw_statement_new(ctx, error, DRAW_STMT_PRIMITIVE);
    if (statement == NULL) {
        return NULL;
    }
    snprintf(statement->primitive, sizeof(statement->primitive), "%s", kind);
    statement->expr_count = count;
    statement->exprs[0] = a;
    statement->exprs[1] = b;
    statement->exprs[2] = c;
    statement->exprs[3] = d;
    return statement;
}

static draw_value draw_default_reduce(const draw_reduction *ctx) {
    if (ctx->rhs_count == 1) {
        return ctx->values[0];
    }
    return NULL;
}

/* Reducer callbacks are the bridge between generated parser tables and
 * application semantics. Each DRAW_ACTION_* label comes from draw.lf and is
 * mapped here to a concrete AST node or helper value. */
static draw_value draw_reduce(const draw_reduction *ctx, void *user, draw_error *error) {
    draw_context *draw = (draw_context *)user;
    switch (ctx->action_id) {
    case DRAW_ACTION_PROGRAM: {
        draw_program *program = (draw_program *)draw_alloc(draw, sizeof(draw_program), error, "program");
        if (program != NULL) {
            program->statements = (draw_statement_list *)ctx->values[0];
        }
        return program;
    }
    case DRAW_ACTION_STATEMENTS:
    case DRAW_ACTION_FIGURES:
        return draw_statement_list_prepend(draw, error, (draw_statement *)ctx->values[0], (draw_statement_list *)ctx->values[1]);
    case DRAW_ACTION_STATEMENT_TAIL_MORE:
    case DRAW_ACTION_FIGURE_TAIL_MORE:
        return draw_statement_list_prepend(draw, error, (draw_statement *)ctx->values[1], (draw_statement_list *)ctx->values[2]);
    case DRAW_ACTION_STATEMENT_TAIL_EMPTY:
    case DRAW_ACTION_FIGURE_TAIL_EMPTY:
        return draw_statement_list_empty(draw, error);
    case DRAW_ACTION_PASS:
        return ctx->values[0];
    case DRAW_ACTION_CANVAS: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_CANVAS);
        if (statement != NULL) {
            statement->exprs[0] = (draw_expr *)ctx->values[1];
            statement->exprs[1] = (draw_expr *)ctx->values[3];
            statement->expr_count = 2;
        }
        return statement;
    }
    case DRAW_ACTION_BACKGROUND:
    case DRAW_ACTION_STROKE: {
        draw_statement *statement = draw_statement_new(draw, error, ctx->action_id == DRAW_ACTION_BACKGROUND ? DRAW_STMT_BACKGROUND : DRAW_STMT_STROKE);
        if (statement != NULL) {
            statement->color = *((draw_color *)ctx->values[1]);
        }
        return statement;
    }
    case DRAW_ACTION_FILL: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_FILL);
        if (statement != NULL) {
            statement->color = *((draw_color *)ctx->values[1]);
            statement->enabled = 1;
        }
        return statement;
    }
    case DRAW_ACTION_FILL_NONE: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_FILL);
        if (statement != NULL) {
            statement->enabled = 0;
        }
        return statement;
    }
    case DRAW_ACTION_WIDTH: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_WIDTH);
        if (statement != NULL) {
            statement->exprs[0] = (draw_expr *)ctx->values[1];
            statement->expr_count = 1;
        }
        return statement;
    }
    case DRAW_ACTION_ASSIGN: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_ASSIGN);
        if (statement != NULL) {
            statement->name = draw_copy_lexeme(draw, draw_lexeme_value(ctx->values[0]), error);
            statement->exprs[0] = (draw_expr *)ctx->values[2];
            statement->expr_count = 1;
        }
        return statement;
    }
    case DRAW_ACTION_DEFINE_FIGURE: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_DEFINE_FIGURE);
        if (statement != NULL) {
            statement->name = draw_copy_lexeme(draw, draw_lexeme_value(ctx->values[0]), error);
            statement->figure = (draw_figure_block *)ctx->values[2];
        }
        return statement;
    }
    case DRAW_ACTION_DRAW: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_DRAW);
        if (statement != NULL) {
            statement->target = (draw_figure_ref *)ctx->values[1];
        }
        return statement;
    }
    case DRAW_ACTION_REPDRAW: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_REPDRAW);
        if (statement != NULL) {
            statement->exprs[0] = (draw_expr *)ctx->values[1];
            statement->expr_count = 1;
            statement->target = (draw_figure_ref *)ctx->values[2];
        }
        return statement;
    }
    case DRAW_ACTION_FIGURE_REF_NAMED: {
        draw_figure_ref *ref = (draw_figure_ref *)draw_alloc(draw, sizeof(draw_figure_ref), error, "figure reference");
        if (ref != NULL) {
            ref->kind = DRAW_FIGURE_NAMED;
            ref->name = draw_copy_lexeme(draw, draw_lexeme_value(ctx->values[0]), error);
        }
        return ref;
    }
    case DRAW_ACTION_FIGURE_REF_INLINE: {
        draw_figure_ref *ref = (draw_figure_ref *)draw_alloc(draw, sizeof(draw_figure_ref), error, "figure reference");
        if (ref != NULL) {
            ref->kind = DRAW_FIGURE_INLINE;
            ref->block = (draw_figure_block *)ctx->values[0];
        }
        return ref;
    }
    case DRAW_ACTION_FIGUREBLOCK: {
        draw_figure_block *block = (draw_figure_block *)draw_alloc(draw, sizeof(draw_figure_block), error, "figure block");
        if (block != NULL) {
            block->statements = (draw_statement_list *)ctx->values[1];
        }
        return block;
    }
    case DRAW_ACTION_PRIMITIVE_POINT:
        return draw_primitive(draw, error, "point", 2, (draw_expr *)ctx->values[1], (draw_expr *)ctx->values[3], NULL, NULL);
    case DRAW_ACTION_PRIMITIVE_LINE:
        return draw_primitive(draw, error, "line", 4, (draw_expr *)ctx->values[1], (draw_expr *)ctx->values[3], (draw_expr *)ctx->values[5], (draw_expr *)ctx->values[7]);
    case DRAW_ACTION_PRIMITIVE_BOX:
        return draw_primitive(draw, error, "box", 4, (draw_expr *)ctx->values[1], (draw_expr *)ctx->values[3], (draw_expr *)ctx->values[5], (draw_expr *)ctx->values[7]);
    case DRAW_ACTION_PRIMITIVE_CIRCLE:
        return draw_primitive(draw, error, "circle", 3, (draw_expr *)ctx->values[1], (draw_expr *)ctx->values[3], (draw_expr *)ctx->values[5], NULL);
    case DRAW_ACTION_COLOR:
        return draw_parse_color(draw, error, draw_lexeme_value(ctx->values[0]));
    case DRAW_ACTION_EXPR:
    case DRAW_ACTION_TERM:
        return draw_fold_binary(draw, error, (draw_expr *)ctx->values[0], (draw_binary_tail_list *)ctx->values[1]);
    case DRAW_ACTION_EXPR_TAIL_ADD:
        return draw_tail_list_prepend(draw, error, '+', (draw_expr *)ctx->values[1], (draw_binary_tail_list *)ctx->values[2]);
    case DRAW_ACTION_EXPR_TAIL_SUBTRACT:
        return draw_tail_list_prepend(draw, error, '-', (draw_expr *)ctx->values[1], (draw_binary_tail_list *)ctx->values[2]);
    case DRAW_ACTION_TERM_TAIL_MULTIPLY:
        return draw_tail_list_prepend(draw, error, '*', (draw_expr *)ctx->values[1], (draw_binary_tail_list *)ctx->values[2]);
    case DRAW_ACTION_TERM_TAIL_DIVIDE:
        return draw_tail_list_prepend(draw, error, '/', (draw_expr *)ctx->values[1], (draw_binary_tail_list *)ctx->values[2]);
    case DRAW_ACTION_EXPR_TAIL_EMPTY:
    case DRAW_ACTION_TERM_TAIL_EMPTY:
        return draw_tail_list_empty(draw, error);
    case DRAW_ACTION_UNARY_NEGATE:
        return draw_expr_unary(draw, error, '-', (draw_expr *)ctx->values[1]);
    case DRAW_ACTION_NUMBER: {
        const draw_lexeme *lexeme = draw_lexeme_value(ctx->values[0]);
        char *text = draw_copy_lexeme(draw, lexeme, error);
        return text == NULL ? NULL : draw_expr_number(draw, error, strtod(text, NULL));
    }
    case DRAW_ACTION_VARIABLE:
        return draw_expr_variable(draw, error, draw_copy_lexeme(draw, draw_lexeme_value(ctx->values[0]), error));
    case DRAW_ACTION_CALL:
        return draw_expr_call(draw, error, draw_copy_lexeme(draw, draw_lexeme_value(ctx->values[0]), error), (draw_expr *)ctx->values[2]);
    case DRAW_ACTION_GROUP:
        return ctx->values[1];
    case DRAW_ACTION_NONE:
    default:
        return draw_default_reduce(ctx);
    }
}

/* The parser remains reentrant because all semantic allocations live in the
 * caller-owned draw_context arena, and no generated or handwritten parser state
 * is stored globally. */
static int draw_parse_source(draw_context *ctx, const char *source, draw_program **out, char *message, size_t message_size) {
    draw_error error;
    draw_lexeme *tokens = NULL;
    draw_value value = NULL;
    size_t count = 0;
    error.message[0] = '\0';
    if (!draw_tokenize(source, &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "scan failed: %s", error.message);
    }
    if (!draw_parse_value(tokens, count, draw_reduce, ctx, &value, &error)) {
        draw_free_lexemes(tokens);
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    draw_free_lexemes(tokens);
    *out = (draw_program *)value;
    return 1;
}

static void draw_renderer_init(draw_renderer *renderer) {
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

static size_t draw_var_count(draw_renderer *renderer) {
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

static size_t draw_figure_count(draw_renderer *renderer) {
    size_t count = 0;
    draw_named_figure *item = renderer->figures;
    while (item != NULL) {
        count++;
        item = item->next;
    }
    return count;
}

static void draw_renderer_free(draw_renderer *renderer) {
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

/* Rendering is deliberately separated from parsing: first reductions build a
 * complete AST, then this interpreter evaluates variables, figures, loops, and
 * drawing primitives against a local image buffer. */
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

static int draw_render(draw_program *program, draw_renderer *renderer, char *message, size_t message_size) {
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

static int draw_compare_names(const void *left, const void *right) {
    const char *const *a = (const char *const *)left;
    const char *const *b = (const char *const *)right;
    return strcmp(*a, *b);
}

static char *draw_color_hex(draw_color color, char *buffer, size_t size) {
    snprintf(buffer, size, "#%02X%02X%02X", color.r, color.g, color.b);
    return buffer;
}

static int draw_append_figures(draw_renderer *renderer, demo_text *report, char *message, size_t message_size) {
    size_t count = draw_figure_count(renderer);
    const char **names = NULL;
    size_t index = 0;
    draw_named_figure *figure = renderer->figures;
    if (count == 0) {
        return demo_text_append(report, "[]", message, message_size);
    }
    names = (const char **)calloc(count, sizeof(char *));
    if (names == NULL) {
        return demo_set_error(message, message_size, "out of memory sorting figure names");
    }
    while (figure != NULL) {
        names[index++] = figure->name;
        figure = figure->next;
    }
    qsort(names, count, sizeof(char *), draw_compare_names);
    if (!demo_text_append(report, "[", message, message_size)) {
        free(names);
        return 0;
    }
    for (index = 0; index < count; index++) {
        if (!demo_text_appendf(report, message, message_size, "%s%s", index == 0 ? "" : ", ", names[index])) {
            free(names);
            return 0;
        }
    }
    free(names);
    return demo_text_append(report, "]", message, message_size);
}

static int draw_append_count(demo_text *report, char *message, size_t message_size, const char *name, int count) {
    if (count <= 0) {
        return 1;
    }
    return demo_text_appendf(report, message, message_size, "  %s: %d\n", name, count);
}

static int draw_append_define_counts(draw_renderer *renderer, demo_text *report, char *message, size_t message_size) {
    size_t count = draw_figure_count(renderer);
    const char **names = NULL;
    size_t index = 0;
    draw_named_figure *figure = renderer->figures;
    if (count == 0) {
        return 1;
    }
    names = (const char **)calloc(count, sizeof(char *));
    if (names == NULL) {
        return demo_set_error(message, message_size, "out of memory sorting figure names");
    }
    while (figure != NULL) {
        names[index++] = figure->name;
        figure = figure->next;
    }
    qsort(names, count, sizeof(char *), draw_compare_names);
    for (index = 0; index < count; index++) {
        if (!demo_text_appendf(report, message, message_size, "  define %s: 1\n", names[index])) {
            free(names);
            return 0;
        }
    }
    free(names);
    return 1;
}

static int draw_build_report(draw_renderer *renderer, const char *input_path, const char *output_path, demo_text *report, char *message, size_t message_size) {
    char background[16];
    char background_label[32];
    char canvas_label[32];
    snprintf(background_label, sizeof(background_label), "background %s", draw_color_hex(renderer->background, background, sizeof(background)));
    snprintf(canvas_label, sizeof(canvas_label), "canvas %d,%d", renderer->image.width, renderer->image.height);
    if (!demo_text_appendf(report, message, message_size,
            "DRAW C render report\nSource: %s\nOutput: %s\nCanvas: %dx%d\nFigures: ",
            input_path,
            output_path,
            renderer->image.width,
            renderer->image.height) ||
        !draw_append_figures(renderer, report, message, message_size) ||
        !demo_text_appendf(report, message, message_size, "\nVariables: %lu\n\nOperation summary:\n", (unsigned long)draw_var_count(renderer)) ||
        !draw_append_count(report, message, message_size, background_label, renderer->counts.background) ||
        !draw_append_count(report, message, message_size, "box 4 args", renderer->counts.box) ||
        !draw_append_count(report, message, message_size, canvas_label, renderer->counts.canvas) ||
        !draw_append_count(report, message, message_size, "circle 3 args", renderer->counts.circle) ||
        !draw_append_define_counts(renderer, report, message, message_size) ||
        !draw_append_count(report, message, message_size, "line 4 args", renderer->counts.line) ||
        !draw_append_count(report, message, message_size, "point 2 args", renderer->counts.point)) {
        return 0;
    }
    return 1;
}

static int draw_render_source(draw_context *ctx, const char *source, const char *input_path, const char *output_path, draw_renderer *renderer, demo_text *report, char *message, size_t message_size) {
    draw_program *program = NULL;
    if (!draw_parse_source(ctx, source, &program, message, message_size) ||
        !draw_render(program, renderer, message, message_size) ||
        !demo_write_png(output_path, &renderer->image, message, message_size) ||
        !draw_build_report(renderer, input_path, output_path, report, message, message_size)) {
        return 0;
    }
    return 1;
}

static int draw_run_assertions(const char *source, const char *output_path, char *message, size_t message_size) {
    draw_context ctx = {0};
    draw_renderer renderer;
    demo_text report = {0};
    draw_error error;
    draw_lexeme *tokens = NULL;
    size_t count = 0;
    error.message[0] = '\0';
    if (!draw_render_source(&ctx, source, "sample.draw", output_path, &renderer, &report, message, message_size)) {
        demo_text_free(&report);
        demo_arena_free(&ctx.arena);
        return 0;
    }
    if (renderer.image.width != 960 || renderer.image.height != 640 || renderer.counts.line != 90 || renderer.counts.circle != 196 || renderer.counts.box != 2) {
        draw_renderer_free(&renderer);
        demo_text_free(&report);
        demo_arena_free(&ctx.arena);
        return demo_set_error(message, message_size, "unexpected draw render summary");
    }
    draw_renderer_free(&renderer);
    demo_text_free(&report);
    demo_arena_free(&ctx.arena);
    if (draw_tokenize("canvas 1, @", &tokens, &count, &error)) {
        draw_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure");
    }
    if (!draw_tokenize("draw ;", &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (draw_parse(tokens, count, &error)) {
        draw_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure");
    }
    draw_free_lexemes(tokens);
    return 1;
}

static const char *draw_read_option(int *argc, char **argv, const char *name, const char *fallback) {
    int i = 1;
    for (i = 1; i + 1 < *argc; i++) {
        if (strcmp(argv[i], name) == 0) {
            const char *value = argv[i + 1];
            int j = i;
            for (j = i; j + 2 < *argc; j++) {
                argv[j] = argv[j + 2];
            }
            *argc -= 2;
            return value;
        }
    }
    return fallback;
}

static int draw_take_flag(int *argc, char **argv, const char *name) {
    int i = 1;
    for (i = 1; i < *argc; i++) {
        if (strcmp(argv[i], name) == 0) {
            int j = i;
            for (j = i; j + 1 < *argc; j++) {
                argv[j] = argv[j + 1];
            }
            *argc -= 1;
            return 1;
        }
    }
    return 0;
}

int main(int argc, char **argv) {
    char message[512] = {0};
    demo_buffer source = {0};
    demo_text report = {0};
    draw_context ctx = {0};
    draw_renderer renderer;
    int assert_mode = draw_take_flag(&argc, argv, "--assert");
    const char *output_path = draw_read_option(&argc, argv, "--output", "dist/sample-c.png");
    const char *log_path = draw_read_option(&argc, argv, "--log", "dist/draw-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "sample.draw";
    if (!demo_read_file(input_path, &source, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (assert_mode && !draw_run_assertions(source.data, output_path, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        return 1;
    }
    if (!draw_render_source(&ctx, source.data, input_path, output_path, &renderer, &report, message, sizeof(message)) ||
        !demo_write_text(log_path, report.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        demo_text_free(&report);
        demo_arena_free(&ctx.arena);
        return 1;
    }
    printf("%s", report.data);
    draw_renderer_free(&renderer);
    demo_free_buffer(&source);
    demo_text_free(&report);
    demo_arena_free(&ctx.arena);
    return 0;
}
