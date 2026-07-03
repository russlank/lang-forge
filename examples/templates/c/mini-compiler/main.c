#include "generated/parser.h"

#include <errno.h>
#include <limits.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef struct expr expr;
typedef struct statement statement;
typedef struct program program;

#include "generated/parser_typed.h"

typedef enum expr_kind {
    EXPR_NUMBER,
    EXPR_ADD
} expr_kind;

struct expr {
    expr_kind kind;
    int value;
    struct expr *left;
    struct expr *right;
};

struct statement {
    expr *value;
    struct statement *next;
};

struct program {
    statement *statements;
};

typedef struct instruction {
    const char *op;
    int arg;
} instruction;

typedef struct code {
    instruction items[64];
    size_t count;
} code;

typedef struct runtime_output {
    int values[16];
    size_t count;
} runtime_output;

typedef struct arena {
    void *items[128];
    size_t count;
} arena;

typedef struct context {
    arena memory;
} context;

static void *arena_alloc(arena *memory, size_t size) {
    void *item = calloc(1, size);
    if (item == NULL || memory->count >= sizeof(memory->items) / sizeof(memory->items[0])) {
        free(item);
        return NULL;
    }
    memory->items[memory->count++] = item;
    return item;
}

static void arena_free(arena *memory) {
    size_t i = 0;
    for (i = 0; i < memory->count; i++) {
        free(memory->items[i]);
    }
    memory->count = 0;
}

static char *read_file(const char *path) {
    FILE *file = fopen(path, "rb");
    long size = 0;
    char *data = NULL;
    if (file == NULL) {
        return NULL;
    }
    if (fseek(file, 0, SEEK_END) != 0 || (size = ftell(file)) < 0 || fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        return NULL;
    }
    data = (char *)calloc((size_t)size + 1, 1);
    if (data == NULL) {
        fclose(file);
        return NULL;
    }
    if (fread(data, 1, (size_t)size, file) != (size_t)size) {
        free(data);
        fclose(file);
        return NULL;
    }
    fclose(file);
    return data;
}

static int write_file(const char *path, const char *text) {
    FILE *file = fopen(path, "wb");
    if (file == NULL) {
        return 0;
    }
    fputs(text, file);
    fclose(file);
    return 1;
}

static expr *new_number(context *ctx, int value) {
    expr *node = (expr *)arena_alloc(&ctx->memory, sizeof(expr));
    if (node != NULL) {
        node->kind = EXPR_NUMBER;
        node->value = value;
    }
    return node;
}

static expr *new_add(context *ctx, expr *left, expr *right) {
    expr *node = (expr *)arena_alloc(&ctx->memory, sizeof(expr));
    if (node != NULL) {
        node->kind = EXPR_ADD;
        node->left = left;
        node->right = right;
    }
    return node;
}

static statement *new_statement(context *ctx, expr *value) {
    statement *node = (statement *)arena_alloc(&ctx->memory, sizeof(statement));
    if (node != NULL) {
        node->value = value;
    }
    return node;
}

static program *new_program(context *ctx, statement *statements) {
    program *node = (program *)arena_alloc(&ctx->memory, sizeof(program));
    if (node != NULL) {
        node->statements = statements;
    }
    return node;
}

static mini_compiler_value reduce_program(const mini_compiler_program_reduction *ctx, void *user, mini_compiler_error *error) {
    context *state = (context *)user;
    (void)error;
    return new_program(state, ctx->statements);
}

static mini_compiler_value reduce_statements(const mini_compiler_statements_reduction *ctx, void *user, mini_compiler_error *error) {
    (void)user;
    if (ctx->head == NULL) {
        snprintf(error->message, sizeof(error->message), "statements reduction missing head statement");
        return NULL;
    }
    ctx->head->next = ctx->tail;
    return ctx->head;
}

static mini_compiler_value reduce_statements_tail_more(const mini_compiler_statements_tail_more_reduction *ctx, void *user, mini_compiler_error *error) {
    (void)user;
    if (ctx->head == NULL) {
        snprintf(error->message, sizeof(error->message), "statements tail reduction missing head statement");
        return NULL;
    }
    ctx->head->next = ctx->tail;
    return ctx->head;
}

static mini_compiler_value reduce_statements_tail_empty(const mini_compiler_statements_tail_empty_reduction *ctx, void *user, mini_compiler_error *error) {
    (void)ctx;
    (void)user;
    (void)error;
    return NULL;
}

static mini_compiler_value reduce_print(const mini_compiler_print_reduction *ctx, void *user, mini_compiler_error *error) {
    context *state = (context *)user;
    (void)error;
    return new_statement(state, ctx->expr);
}

static mini_compiler_value reduce_add(const mini_compiler_add_reduction *ctx, void *user, mini_compiler_error *error) {
    context *state = (context *)user;
    (void)error;
    return new_add(state, ctx->left, ctx->right);
}

static mini_compiler_value reduce_pass(const mini_compiler_pass_reduction *ctx, void *user, mini_compiler_error *error) {
    (void)user;
    (void)error;
    return ctx->value;
}

static mini_compiler_value reduce_number(const mini_compiler_number_reduction *ctx, void *user, mini_compiler_error *error) {
    context *state = (context *)user;
    char text[32] = {0};
    char *end = NULL;
    long value = 0;
    if (ctx->token == NULL || ctx->token->length >= sizeof(text)) {
        snprintf(error->message, sizeof(error->message), "rule %d action number label token has invalid number literal", ctx->reduction->rule);
        return NULL;
    }
    memcpy(text, ctx->token->text, ctx->token->length);
    errno = 0;
    value = strtol(text, &end, 10);
    if (errno == ERANGE || end == text || *end != '\0' || value < INT_MIN || value > INT_MAX) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label token value %s is not a valid int", ctx->reduction->rule, ctx->reduction->action, text);
        return NULL;
    }
    return new_number(state, (int)value);
}

static mini_compiler_typed_reducer make_typed_reducer(context *state) {
    /*
     * Each handler receives a generated typed context. For example, the
     * `Expr : left=Expr Plus right=Term {c: add}` rule becomes fields named
     * `left` and `right`, avoiding positional `ctx->values[index]` access.
     */
    mini_compiler_typed_reducer reducer;
    reducer.user = state;
    reducer.program = reduce_program;
    reducer.statements = reduce_statements;
    reducer.statements_tail_more = reduce_statements_tail_more;
    reducer.statements_tail_empty = reduce_statements_tail_empty;
    reducer.print = reduce_print;
    reducer.add = reduce_add;
    reducer.pass = reduce_pass;
    reducer.number = reduce_number;
    return reducer;
}

static program *parse_source(context *state, const char *source, char *message, size_t message_size) {
    mini_compiler_error error;
    mini_compiler_scanner scanner;
    mini_compiler_lexeme_source token_source;
    mini_compiler_typed_reducer reducer;
    mini_compiler_value value = NULL;
    error.message[0] = '\0';
    mini_compiler_scanner_init(&scanner, source);
    token_source.user = &scanner;
    token_source.next = mini_compiler_scanner_source_next;
    reducer = make_typed_reducer(state);
    if (!mini_compiler_parse_value_source_typed(&token_source, &reducer, &value, &error)) {
        snprintf(message, message_size, "parse failed: %s", error.message);
        return NULL;
    }
    return (program *)value;
}

static int emit(code *out, const char *op, int arg) {
    if (out->count >= sizeof(out->items) / sizeof(out->items[0])) {
        return 0;
    }
    out->items[out->count].op = op;
    out->items[out->count].arg = arg;
    out->count++;
    return 1;
}

static int compile_expr(expr *node, code *out) {
    if (node->kind == EXPR_NUMBER) {
        return emit(out, "push", node->value);
    }
    return compile_expr(node->left, out) && compile_expr(node->right, out) && emit(out, "add", 0);
}

static int compile_program(program *node, code *out) {
    statement *stmt = node->statements;
    while (stmt != NULL) {
        if (!compile_expr(stmt->value, out) || !emit(out, "print", 0)) {
            return 0;
        }
        stmt = stmt->next;
    }
    return 1;
}

static int run_code(const code *in, runtime_output *output) {
    int stack[32] = {0};
    size_t depth = 0;
    size_t pc = 0;
    for (pc = 0; pc < in->count; pc++) {
        instruction inst = in->items[pc];
        if (strcmp(inst.op, "push") == 0) {
            stack[depth++] = inst.arg;
        } else if (strcmp(inst.op, "add") == 0) {
            stack[depth - 2] = stack[depth - 2] + stack[depth - 1];
            depth--;
        } else if (strcmp(inst.op, "print") == 0) {
            output->values[output->count++] = stack[--depth];
        }
    }
    return 1;
}

static void build_report(char *buffer, size_t size, const char *input_path, const code *compiled, const runtime_output *output) {
    size_t used = 0;
    size_t i = 0;
    used += (size_t)snprintf(buffer + used, size - used, "Mini compiler C template: %s\nstack code:\n", input_path);
    for (i = 0; i < compiled->count; i++) {
        if (strcmp(compiled->items[i].op, "push") == 0) {
            used += (size_t)snprintf(buffer + used, size - used, "  %02zu push %d\n", i, compiled->items[i].arg);
        } else {
            used += (size_t)snprintf(buffer + used, size - used, "  %02zu %s\n", i, compiled->items[i].op);
        }
    }
    used += (size_t)snprintf(buffer + used, size - used, "output:");
    for (i = 0; i < output->count; i++) {
        used += (size_t)snprintf(buffer + used, size - used, " %d", output->values[i]);
    }
    snprintf(buffer + used, size - used, "\n");
}

static const char *read_option(int *argc, char **argv, const char *name, const char *fallback) {
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

static int take_flag(int *argc, char **argv, const char *name) {
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

static int assert_reducer_error(char *message, size_t message_size) {
    context bad = {0};
    char local[256] = {0};
    program *parsed = parse_source(&bad, "print 999999999999999999999999;", local, sizeof(local));
    arena_free(&bad.memory);
    if (parsed != NULL) {
        snprintf(message, message_size, "expected reducer failure for oversized number");
        return 0;
    }
    if (strstr(local, "action number") == NULL || strstr(local, "label token") == NULL) {
        snprintf(message, message_size, "wrong reducer error: %s", local);
        return 0;
    }
    return 1;
}

int main(int argc, char **argv) {
    char message[256] = {0};
    char report[2048] = {0};
    context state = {0};
    code compiled = {0};
    runtime_output output = {0};
    int assert_mode = take_flag(&argc, argv, "--assert");
    const char *log_path = read_option(&argc, argv, "--log", "dist/mini-c.log");
    const char *input_path = argc > 1 ? argv[1] : "input.mini";
    char *source = read_file(input_path);
    program *parsed = NULL;
    if (source == NULL) {
        fprintf(stderr, "cannot read %s\n", input_path);
        return 1;
    }
    parsed = parse_source(&state, source, message, sizeof(message));
    if (parsed == NULL || !compile_program(parsed, &compiled) || !run_code(&compiled, &output)) {
        fprintf(stderr, "%s\n", message[0] == '\0' ? "mini compiler failed" : message);
        free(source);
        arena_free(&state.memory);
        return 1;
    }
    if (assert_mode && (output.count != 2 || output.values[0] != 3 || output.values[1] != 42)) {
        fprintf(stderr, "unexpected template output\n");
        free(source);
        arena_free(&state.memory);
        return 1;
    }
    if (assert_mode && !assert_reducer_error(message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        free(source);
        arena_free(&state.memory);
        return 1;
    }
    build_report(report, sizeof(report), input_path, &compiled, &output);
    printf("%s", report);
    if (!write_file(log_path, report)) {
        fprintf(stderr, "cannot write %s\n", log_path);
        free(source);
        arena_free(&state.memory);
        return 1;
    }
    free(source);
    arena_free(&state.memory);
    return 0;
}
