#include "parser_adapter.h"

#include "generated/parser.h"
#include "generated/parser_typed.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void *draw_alloc(draw_context *ctx, size_t size, draw_error *error, const char *what) {
    void *ptr = demo_arena_alloc(&ctx->arena, size);
    if (ptr == NULL) {
        snprintf(error->message, sizeof(error->message), "out of memory allocating %s", what);
    }
    return ptr;
}

static draw_value draw_arg_value(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    /*
     * The C backend currently passes boxed semantic values. This helper keeps
     * positional access in one place while call sites use grammar-oriented
     * names such as "figure reference" and "right expression".
     */
    if (index >= ctx->rhs_count) {
        snprintf(error->message, sizeof(error->message),
            "rule %d action %s is missing %s at argument %lu",
            ctx->rule,
            ctx->action,
            name,
            (unsigned long)(index + 1));
        return NULL;
    }
    if (ctx->values[index] == NULL) {
        snprintf(error->message, sizeof(error->message),
            "rule %d action %s has null %s at argument %lu",
            ctx->rule,
            ctx->action,
            name,
            (unsigned long)(index + 1));
        return NULL;
    }
    return ctx->values[index];
}

static const draw_lexeme *draw_arg_lexeme(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (const draw_lexeme *)draw_arg_value(ctx, index, name, error);
}

static draw_expr *draw_arg_expr(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_expr *)draw_arg_value(ctx, index, name, error);
}

static draw_statement *draw_arg_statement(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_statement *)draw_arg_value(ctx, index, name, error);
}

static draw_statement_list *draw_arg_statement_list(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_statement_list *)draw_arg_value(ctx, index, name, error);
}

static draw_binary_tail_list *draw_arg_tail_list(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_binary_tail_list *)draw_arg_value(ctx, index, name, error);
}

static draw_color *draw_arg_color(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_color *)draw_arg_value(ctx, index, name, error);
}

static draw_figure_block *draw_arg_figure_block(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_figure_block *)draw_arg_value(ctx, index, name, error);
}

static draw_figure_ref *draw_arg_figure_ref(const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    return (draw_figure_ref *)draw_arg_value(ctx, index, name, error);
}

static char *draw_copy_lexeme(draw_context *ctx, const draw_lexeme *lexeme, draw_error *error) {
    char *copy = demo_arena_copy(&ctx->arena, lexeme->text, lexeme->length);
    if (copy == NULL) {
        snprintf(error->message, sizeof(error->message), "out of memory copying lexeme text");
    }
    return copy;
}

static char *draw_copy_arg_text(draw_context *draw, const draw_reduction *ctx, size_t index, const char *name, draw_error *error) {
    const draw_lexeme *lexeme = draw_arg_lexeme(ctx, index, name, error);
    if (lexeme == NULL) {
        return NULL;
    }
    return draw_copy_lexeme(draw, lexeme, error);
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
    draw_color *color = NULL;
    if (lexeme == NULL) {
        return NULL;
    }
    if (lexeme->length != 7 || lexeme->text[0] != '#') {
        snprintf(error->message, sizeof(error->message), "invalid color literal");
        return NULL;
    }
    color = (draw_color *)draw_alloc(ctx, sizeof(draw_color), error, "color");
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

/* Reducer callbacks are the only place where generated reduction positions are
 * interpreted. The typed helpers above keep casts and positional knowledge out
 * of the renderer and CLI. */
static draw_value draw_reduce(const draw_reduction *ctx, void *user, draw_error *error) {
    draw_context *draw = (draw_context *)user;
    switch (ctx->action_id) {
    case DRAW_ACTION_PROGRAM: {
        draw_program *program = (draw_program *)draw_alloc(draw, sizeof(draw_program), error, "program");
        if (program != NULL) {
            program->statements = draw_arg_statement_list(ctx, 0, "statement list", error);
        }
        return program;
    }
    case DRAW_ACTION_STATEMENTS:
    case DRAW_ACTION_FIGURES:
        return draw_statement_list_prepend(draw, error, draw_arg_statement(ctx, 0, "statement", error), draw_arg_statement_list(ctx, 1, "tail statements", error));
    case DRAW_ACTION_STATEMENT_TAIL_MORE:
    case DRAW_ACTION_FIGURE_TAIL_MORE:
        return draw_statement_list_prepend(draw, error, draw_arg_statement(ctx, 1, "statement", error), draw_arg_statement_list(ctx, 2, "tail statements", error));
    case DRAW_ACTION_STATEMENT_TAIL_EMPTY:
    case DRAW_ACTION_FIGURE_TAIL_EMPTY:
        return draw_statement_list_empty(draw, error);
    case DRAW_ACTION_PASS:
        return draw_arg_value(ctx, 0, "pass-through value", error);
    case DRAW_ACTION_CANVAS: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_CANVAS);
        if (statement != NULL) {
            statement->exprs[0] = draw_arg_expr(ctx, 1, "canvas width", error);
            statement->exprs[1] = draw_arg_expr(ctx, 3, "canvas height", error);
            statement->expr_count = 2;
        }
        return statement;
    }
    case DRAW_ACTION_BACKGROUND:
    case DRAW_ACTION_STROKE: {
        draw_statement *statement = draw_statement_new(draw, error, ctx->action_id == DRAW_ACTION_BACKGROUND ? DRAW_STMT_BACKGROUND : DRAW_STMT_STROKE);
        draw_color *color = draw_arg_color(ctx, 1, "color", error);
        if (statement != NULL && color != NULL) {
            statement->color = *color;
        }
        return statement;
    }
    case DRAW_ACTION_FILL: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_FILL);
        draw_color *color = draw_arg_color(ctx, 1, "fill color", error);
        if (statement != NULL && color != NULL) {
            statement->color = *color;
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
            statement->exprs[0] = draw_arg_expr(ctx, 1, "line width", error);
            statement->expr_count = 1;
        }
        return statement;
    }
    case DRAW_ACTION_ASSIGN: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_ASSIGN);
        if (statement != NULL) {
            statement->name = draw_copy_arg_text(draw, ctx, 0, "variable name", error);
            statement->exprs[0] = draw_arg_expr(ctx, 2, "assigned expression", error);
            statement->expr_count = 1;
        }
        return statement;
    }
    case DRAW_ACTION_DEFINE_FIGURE: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_DEFINE_FIGURE);
        if (statement != NULL) {
            statement->name = draw_copy_arg_text(draw, ctx, 0, "figure name", error);
            statement->figure = draw_arg_figure_block(ctx, 2, "figure block", error);
        }
        return statement;
    }
    case DRAW_ACTION_DRAW: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_DRAW);
        if (statement != NULL) {
            statement->target = draw_arg_figure_ref(ctx, 1, "figure reference", error);
        }
        return statement;
    }
    case DRAW_ACTION_REPDRAW: {
        draw_statement *statement = draw_statement_new(draw, error, DRAW_STMT_REPDRAW);
        if (statement != NULL) {
            statement->exprs[0] = draw_arg_expr(ctx, 1, "repeat count", error);
            statement->expr_count = 1;
            statement->target = draw_arg_figure_ref(ctx, 2, "figure reference", error);
        }
        return statement;
    }
    case DRAW_ACTION_FIGURE_REF_NAMED: {
        draw_figure_ref *ref = (draw_figure_ref *)draw_alloc(draw, sizeof(draw_figure_ref), error, "figure reference");
        if (ref != NULL) {
            ref->kind = DRAW_FIGURE_NAMED;
            ref->name = draw_copy_arg_text(draw, ctx, 0, "figure name", error);
        }
        return ref;
    }
    case DRAW_ACTION_FIGURE_REF_INLINE: {
        draw_figure_ref *ref = (draw_figure_ref *)draw_alloc(draw, sizeof(draw_figure_ref), error, "figure reference");
        if (ref != NULL) {
            ref->kind = DRAW_FIGURE_INLINE;
            ref->block = draw_arg_figure_block(ctx, 0, "inline figure", error);
        }
        return ref;
    }
    case DRAW_ACTION_FIGURE_BLOCK: {
        draw_figure_block *block = (draw_figure_block *)draw_alloc(draw, sizeof(draw_figure_block), error, "figure block");
        if (block != NULL) {
            block->statements = draw_arg_statement_list(ctx, 1, "figure statements", error);
        }
        return block;
    }
    case DRAW_ACTION_PRIMITIVE_POINT:
        return draw_primitive(draw, error, "point", 2, draw_arg_expr(ctx, 1, "point x", error), draw_arg_expr(ctx, 3, "point y", error), NULL, NULL);
    case DRAW_ACTION_PRIMITIVE_LINE:
        return draw_primitive(draw, error, "line", 4, draw_arg_expr(ctx, 1, "line x1", error), draw_arg_expr(ctx, 3, "line y1", error), draw_arg_expr(ctx, 5, "line x2", error), draw_arg_expr(ctx, 7, "line y2", error));
    case DRAW_ACTION_PRIMITIVE_BOX:
        return draw_primitive(draw, error, "box", 4, draw_arg_expr(ctx, 1, "box x1", error), draw_arg_expr(ctx, 3, "box y1", error), draw_arg_expr(ctx, 5, "box x2", error), draw_arg_expr(ctx, 7, "box y2", error));
    case DRAW_ACTION_PRIMITIVE_CIRCLE:
        return draw_primitive(draw, error, "circle", 3, draw_arg_expr(ctx, 1, "circle x", error), draw_arg_expr(ctx, 3, "circle y", error), draw_arg_expr(ctx, 5, "circle radius", error), NULL);
    case DRAW_ACTION_COLOR:
        return draw_parse_color(draw, error, draw_arg_lexeme(ctx, 0, "color literal", error));
    case DRAW_ACTION_EXPR:
        return draw_fold_binary(draw, error, draw_arg_expr(ctx, 0, "left expression", error), draw_arg_tail_list(ctx, 1, "expression tail", error));
    case DRAW_ACTION_TERM:
        return draw_fold_binary(draw, error, draw_arg_expr(ctx, 0, "left term", error), draw_arg_tail_list(ctx, 1, "term tail", error));
    case DRAW_ACTION_EXPR_TAIL_ADD:
        return draw_tail_list_prepend(draw, error, '+', draw_arg_expr(ctx, 1, "right expression", error), draw_arg_tail_list(ctx, 2, "tail expressions", error));
    case DRAW_ACTION_EXPR_TAIL_SUBTRACT:
        return draw_tail_list_prepend(draw, error, '-', draw_arg_expr(ctx, 1, "right expression", error), draw_arg_tail_list(ctx, 2, "tail expressions", error));
    case DRAW_ACTION_TERM_TAIL_MULTIPLY:
        return draw_tail_list_prepend(draw, error, '*', draw_arg_expr(ctx, 1, "right term", error), draw_arg_tail_list(ctx, 2, "tail terms", error));
    case DRAW_ACTION_TERM_TAIL_DIVIDE:
        return draw_tail_list_prepend(draw, error, '/', draw_arg_expr(ctx, 1, "right term", error), draw_arg_tail_list(ctx, 2, "tail terms", error));
    case DRAW_ACTION_EXPR_TAIL_EMPTY:
    case DRAW_ACTION_TERM_TAIL_EMPTY:
        return draw_tail_list_empty(draw, error);
    case DRAW_ACTION_UNARY_NEGATE:
        return draw_expr_unary(draw, error, '-', draw_arg_expr(ctx, 1, "unary operand", error));
    case DRAW_ACTION_EXPR_PASS:
        return draw_arg_value(ctx, 0, "expression", error);
    case DRAW_ACTION_NUMBER: {
        char *text = draw_copy_arg_text(draw, ctx, 0, "number", error);
        return text == NULL ? NULL : draw_expr_number(draw, error, strtod(text, NULL));
    }
    case DRAW_ACTION_VARIABLE:
        return draw_expr_variable(draw, error, draw_copy_arg_text(draw, ctx, 0, "variable name", error));
    case DRAW_ACTION_CALL:
        return draw_expr_call(draw, error, draw_copy_arg_text(draw, ctx, 0, "function name", error), draw_arg_expr(ctx, 2, "function argument", error));
    case DRAW_ACTION_GROUP:
        return draw_arg_value(ctx, 1, "group expression", error);
    case DRAW_ACTION_NONE:
    default:
        return draw_default_reduce(ctx);
    }
}

int draw_compile_source_with_mode(draw_context *ctx, const char *source, draw_reducer_mode mode, draw_program **out, char *message, size_t message_size) {
    draw_error error;
    draw_scanner scanner;
    draw_lexeme_source lexeme_source;
    draw_value value = NULL;
    int parsed = 0;
    error.message[0] = '\0';
    draw_scanner_init(&scanner, source);
    lexeme_source.user = &scanner;
    lexeme_source.next = draw_scanner_lexeme_source_next;
    if (mode == DRAW_REDUCER_TYPED) {
        draw_boxed_typed_reducer boxed = {0};
        draw_typed_reducer typed = draw_typed_reducer_from_boxed(&boxed, draw_reduce, ctx);
        parsed = draw_parse_value_lexeme_source_typed(&lexeme_source, &typed, &value, &error);
    } else {
        parsed = draw_parse_value_lexeme_source(&lexeme_source, draw_reduce, ctx, &value, &error);
    }
    if (!parsed) {
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    *out = (draw_program *)value;
    return 1;
}

int draw_compile_source(draw_context *ctx, const char *source, draw_program **out, char *message, size_t message_size) {
    return draw_compile_source_with_mode(ctx, source, DRAW_REDUCER_TYPED, out, message, message_size);
}
