#include "demo.h"
#include "parser.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef struct calc_demo {
    demo_arena arena;
} calc_demo;

static calc_value calc_number(calc_demo *demo, calc_error *error, double value) {
    double *slot = (double *)demo_arena_alloc(&demo->arena, sizeof(double));
    if (slot == NULL) {
        snprintf(error->message, sizeof(error->message), "out of memory storing semantic value");
        return NULL;
    }
    *slot = value;
    return slot;
}

static double calc_value_as_number(calc_value value) {
    return *((double *)value);
}

static const calc_lexeme *calc_value_as_lexeme(calc_value value) {
    return (const calc_lexeme *)value;
}

static calc_value calc_reduce(const calc_reduction *ctx, void *user, calc_error *error) {
    calc_demo *demo = (calc_demo *)user;
    switch (ctx->action_id) {
    case CALC_ACTION_START:
    case CALC_ACTION_PASS:
        return ctx->values[0];
    case CALC_ACTION_GROUP:
        return ctx->values[1];
    case CALC_ACTION_NUMBER: {
        const calc_lexeme *lexeme = calc_value_as_lexeme(ctx->values[0]);
        char *text = demo_arena_copy(&demo->arena, lexeme->text, lexeme->length);
        if (text == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory parsing number");
            return NULL;
        }
        return calc_number(demo, error, strtod(text, NULL));
    }
    case CALC_ACTION_NEGATE:
        return calc_number(demo, error, -calc_value_as_number(ctx->values[1]));
    case CALC_ACTION_ADD:
        return calc_number(demo, error, calc_value_as_number(ctx->values[0]) + calc_value_as_number(ctx->values[2]));
    case CALC_ACTION_SUBTRACT:
        return calc_number(demo, error, calc_value_as_number(ctx->values[0]) - calc_value_as_number(ctx->values[2]));
    case CALC_ACTION_MULTIPLY:
        return calc_number(demo, error, calc_value_as_number(ctx->values[0]) * calc_value_as_number(ctx->values[2]));
    case CALC_ACTION_DIVIDE: {
        double right = calc_value_as_number(ctx->values[2]);
        if (right == 0.0) {
            snprintf(error->message, sizeof(error->message), "division by zero");
            return NULL;
        }
        return calc_number(demo, error, calc_value_as_number(ctx->values[0]) / right);
    }
    case CALC_ACTION_NONE:
    default:
        return ctx->rhs_count == 1 ? ctx->values[0] : NULL;
    }
}

static int calc_eval(calc_demo *demo, const char *source, double *out, char *message, size_t message_size) {
    calc_error error;
    calc_lexeme *tokens = NULL;
    size_t count = 0;
    calc_value value = NULL;
    error.message[0] = '\0';
    if (!calc_tokenize(source, &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "scan failed: %s", error.message);
    }
    if (!calc_parse_value(tokens, count, calc_reduce, demo, &value, &error)) {
        calc_free_lexemes(tokens);
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    calc_free_lexemes(tokens);
    *out = calc_value_as_number(value);
    return 1;
}

static int calc_close_enough(double actual, double expected) {
    double delta = actual - expected;
    if (delta < 0) {
        delta = -delta;
    }
    return delta < 0.000001;
}

static int calc_run_assertions(char *message, size_t message_size) {
    calc_demo demo = {0};
    double value = 0.0;
    calc_error error;
    calc_lexeme *tokens = NULL;
    size_t count = 0;
    error.message[0] = '\0';
    if (!calc_eval(&demo, "1+2*(3-4)", &value, message, message_size) || !calc_close_enough(value, -1.0)) {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "calculator assertion failed, got %.6f", value);
    }
    demo_arena_free(&demo.arena);
    if (calc_tokenize("1@", &tokens, &count, &error)) {
        calc_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure for unmatched input");
    }
    if (!calc_tokenize("1+", &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (calc_parse(tokens, count, &error)) {
        calc_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure for incomplete expression");
    }
    calc_free_lexemes(tokens);
    return 1;
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
    char message[512] = {0};
    demo_buffer source = {0};
    demo_text report = {0};
    calc_demo demo = {0};
    double result = 0.0;
    int assert_mode = take_flag(&argc, argv, "--assert");
    const char *log_path = read_option(&argc, argv, "--log", "dist/calc-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "input.calc";
    if (assert_mode && !calc_run_assertions(message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (!demo_read_file(input_path, &source, message, sizeof(message)) ||
        !calc_eval(&demo, source.data, &result, message, sizeof(message)) ||
        !demo_text_appendf(&report, message, sizeof(message), "LangForge C calculator demo\nsource: %s\nresult: %.10g\n", source.data, result) ||
        !demo_write_text(log_path, report.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        demo_text_free(&report);
        demo_arena_free(&demo.arena);
        return 1;
    }
    printf("%s", report.data);
    demo_free_buffer(&source);
    demo_text_free(&report);
    demo_arena_free(&demo.arena);
    return 0;
}
