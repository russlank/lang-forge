#include "generated/parser.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef enum expr_kind {
    EXPR_NUMBER,
    EXPR_ADD
} expr_kind;

typedef struct expr {
    expr_kind kind;
    int value;
    struct expr *left;
    struct expr *right;
} expr;

typedef struct statement {
    expr *value;
    struct statement *next;
} statement;

typedef struct program {
    statement *statements;
} program;

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

static mini_compiler_value value_arg(const mini_compiler_reduction *ctx, size_t index, const char *name, mini_compiler_error *error) {
    if (index >= ctx->rhs_count || ctx->values[index] == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d missing %s at argument %zu", ctx->rule, name, index + 1);
        return NULL;
    }
    return ctx->values[index];
}

static const mini_compiler_lexeme *lexeme_arg(const mini_compiler_reduction *ctx, size_t index, const char *name, mini_compiler_error *error) {
    return (const mini_compiler_lexeme *)value_arg(ctx, index, name, error);
}

static expr *expr_arg(const mini_compiler_reduction *ctx, size_t index, const char *name, mini_compiler_error *error) {
    return (expr *)value_arg(ctx, index, name, error);
}

static statement *statement_arg(const mini_compiler_reduction *ctx, size_t index, const char *name, mini_compiler_error *error) {
    return (statement *)value_arg(ctx, index, name, error);
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

static mini_compiler_value reduce(const mini_compiler_reduction *ctx, void *user, mini_compiler_error *error) {
    context *state = (context *)user;
    switch (ctx->action_id) {
    case MINI_COMPILER_ACTION_PROGRAM:
        return new_program(state, statement_arg(ctx, 0, "statements", error));
    case MINI_COMPILER_ACTION_STATEMENTS: {
        statement *head = statement_arg(ctx, 0, "statement", error);
        head->next = (statement *)ctx->values[1];
        return head;
    }
    case MINI_COMPILER_ACTION_STATEMENTS_TAIL_MORE: {
        statement *head = statement_arg(ctx, 0, "statement", error);
        head->next = (statement *)ctx->values[1];
        return head;
    }
    case MINI_COMPILER_ACTION_STATEMENTS_TAIL_EMPTY:
        return NULL;
    case MINI_COMPILER_ACTION_PRINT:
        return new_statement(state, expr_arg(ctx, 1, "print expression", error));
    case MINI_COMPILER_ACTION_ADD:
        return new_add(state, expr_arg(ctx, 0, "left operand", error), expr_arg(ctx, 2, "right operand", error));
    case MINI_COMPILER_ACTION_PASS:
        return ctx->values[0];
    case MINI_COMPILER_ACTION_NUMBER: {
        const mini_compiler_lexeme *lexeme = lexeme_arg(ctx, 0, "number literal", error);
        char text[32] = {0};
        if (lexeme == NULL || lexeme->length >= sizeof(text)) {
            snprintf(error->message, sizeof(error->message), "invalid number literal");
            return NULL;
        }
        memcpy(text, lexeme->text, lexeme->length);
        return new_number(state, atoi(text));
    }
    case MINI_COMPILER_ACTION_NONE:
    default:
        return ctx->rhs_count == 1 ? ctx->values[0] : NULL;
    }
}

static program *parse_source(context *state, const char *source, char *message, size_t message_size) {
    mini_compiler_error error;
    mini_compiler_scanner scanner;
    mini_compiler_lexeme_source token_source;
    mini_compiler_value value = NULL;
    error.message[0] = '\0';
    mini_compiler_scanner_init(&scanner, source);
    token_source.user = &scanner;
    token_source.next = mini_compiler_scanner_source_next;
    if (!mini_compiler_parse_value_source(&token_source, reduce, state, &value, &error)) {
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
