#include "demo.h"
#include "parser.h"

#include <stdio.h>
#include <string.h>

typedef struct dks_demo {
    demo_arena arena;
    demo_text report;
    int parameters;
    int commands;
} dks_demo;

static char *dks_copy_lexeme(dks_demo *demo, const datakeeper_lexeme *lexeme) {
    return demo_arena_copy(&demo->arena, lexeme->text, lexeme->length);
}

static const datakeeper_lexeme *dks_lexeme(datakeeper_value value) {
    return (const datakeeper_lexeme *)value;
}

static char *dks_decode_literal(dks_demo *demo, const datakeeper_lexeme *lexeme) {
    const char *text = lexeme->text;
    size_t length = lexeme->length;
    char *out = NULL;
    size_t i = 0;
    size_t j = 0;
    if (length >= 4 && text[0] == '#' && text[1] == '{' && text[length - 2] == '#' && text[length - 1] == '}') {
        out = (char *)demo_arena_alloc(&demo->arena, length - 2);
        if (out == NULL) {
            return NULL;
        }
        for (i = 2; i + 2 < length; i++) {
            if (text[i] == '#' && i + 1 < length && text[i + 1] == '#') {
                out[j++] = '#';
                i++;
            } else {
                out[j++] = text[i];
            }
        }
        out[j] = '\0';
        return out;
    }
    if (length >= 2 && text[0] == '"' && text[length - 1] == '"') {
        out = (char *)demo_arena_alloc(&demo->arena, length);
        if (out == NULL) {
            return NULL;
        }
        for (i = 1; i + 1 < length; i++) {
            if (text[i] == '\\' && i + 1 < length - 1) {
                i++;
            }
            out[j++] = text[i];
        }
        out[j] = '\0';
        return out;
    }
    return dks_copy_lexeme(demo, lexeme);
}

static char *dks_ident_value(dks_demo *demo, const datakeeper_lexeme *lexeme) {
    char *name = dks_copy_lexeme(demo, lexeme);
    char *value = NULL;
    if (name == NULL) {
        return NULL;
    }
    value = (char *)demo_arena_alloc(&demo->arena, strlen(name) + 2);
    if (value == NULL) {
        return NULL;
    }
    value[0] = '$';
    strcpy(value + 1, name);
    return value;
}

static int dks_append_parameter(dks_demo *demo, const char *name, datakeeper_error *error) {
    demo->parameters++;
    if (!demo_text_appendf(&demo->report, error->message, sizeof(error->message), "  param %-2d %s\n", demo->parameters, name)) {
        return 0;
    }
    return 1;
}

static int dks_append_command(dks_demo *demo, datakeeper_error *error, const char *kind, const char *a, const char *b, const char *c) {
    demo->commands++;
    if (!demo_text_appendf(&demo->report, error->message, sizeof(error->message), "  %02d %-14s", demo->commands, kind)) {
        return 0;
    }
    if (a != NULL && !demo_text_appendf(&demo->report, error->message, sizeof(error->message), " %s", a)) {
        return 0;
    }
    if (b != NULL && !demo_text_appendf(&demo->report, error->message, sizeof(error->message), " | %s", b)) {
        return 0;
    }
    if (c != NULL && !demo_text_appendf(&demo->report, error->message, sizeof(error->message), " | %s", c)) {
        return 0;
    }
    return demo_text_append(&demo->report, "\n", error->message, sizeof(error->message));
}

static datakeeper_value dks_default_reduce(const datakeeper_reduction *ctx) {
    if (ctx->rhs_count == 1) {
        return ctx->values[0];
    }
    return NULL;
}

static datakeeper_value dks_reduce(const datakeeper_reduction *ctx, void *user, datakeeper_error *error) {
    dks_demo *demo = (dks_demo *)user;
    switch (ctx->action_id) {
    case DATAKEEPER_ACTION_PARAMETERS_DECL: {
        char *name = dks_copy_lexeme(demo, dks_lexeme(ctx->values[0]));
        if (name == NULL || !dks_append_parameter(demo, name, error)) {
            snprintf(error->message, sizeof(error->message), "out of memory collecting parameter");
        }
        return NULL;
    }
    case DATAKEEPER_ACTION_PARAMETERS_TAIL_MORE: {
        char *name = dks_copy_lexeme(demo, dks_lexeme(ctx->values[1]));
        if (name == NULL || !dks_append_parameter(demo, name, error)) {
            snprintf(error->message, sizeof(error->message), "out of memory collecting parameter");
        }
        return NULL;
    }
    case DATAKEEPER_ACTION_VALUE_STRING:
    {
        char *value = dks_decode_literal(demo, dks_lexeme(ctx->values[0]));
        if (value == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory decoding string literal");
        }
        return value;
    }
    case DATAKEEPER_ACTION_VALUE_NUMBER:
    {
        char *value = dks_copy_lexeme(demo, dks_lexeme(ctx->values[0]));
        if (value == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory copying number literal");
        }
        return value;
    }
    case DATAKEEPER_ACTION_VALUE_IDENT:
    {
        char *value = dks_ident_value(demo, dks_lexeme(ctx->values[0]));
        if (value == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory copying identifier value");
        }
        return value;
    }
    case DATAKEEPER_ACTION_ASSIGN: {
        char *name = dks_copy_lexeme(demo, dks_lexeme(ctx->values[0]));
        dks_append_command(demo, error, "assign", name, (const char *)ctx->values[2], NULL);
        return NULL;
    }
    case DATAKEEPER_ACTION_REPLACE:
        dks_append_command(demo, error, "replace", dks_copy_lexeme(demo, dks_lexeme(ctx->values[2])), (const char *)ctx->values[4], (const char *)ctx->values[6]);
        return NULL;
    case DATAKEEPER_ACTION_SQLRUN:
        dks_append_command(demo, error, "sqlrun", (const char *)ctx->values[2], (const char *)ctx->values[4], NULL);
        return NULL;
    case DATAKEEPER_ACTION_ADD_OBJECT:
        dks_append_command(demo, error, "addobject", (const char *)ctx->values[2], (const char *)ctx->values[4], NULL);
        return NULL;
    case DATAKEEPER_ACTION_REMOVE_OBJECT:
        dks_append_command(demo, error, "removeobject", (const char *)ctx->values[2], (const char *)ctx->values[4], NULL);
        return NULL;
    case DATAKEEPER_ACTION_RUN_OBJECTS_JOB:
        dks_append_command(demo, error, "runobjectsjob", (const char *)ctx->values[2], (const char *)ctx->values[4], (const char *)ctx->values[6]);
        return NULL;
    case DATAKEEPER_ACTION_NONE:
    default:
        return dks_default_reduce(ctx);
    }
}

static int dks_parse(dks_demo *demo, const char *source, char *message, size_t message_size) {
    datakeeper_error error;
    datakeeper_lexeme *tokens = NULL;
    size_t count = 0;
    error.message[0] = '\0';
    if (!datakeeper_tokenize(source, &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "scan failed: %s", error.message);
    }
    if (!demo_text_append(&demo->report, "DataKeeper C mock compiler\nparameters:\n", message, message_size) ||
        !datakeeper_parse_value(tokens, count, dks_reduce, demo, NULL, &error)) {
        datakeeper_free_lexemes(tokens);
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    datakeeper_free_lexemes(tokens);
    if (!demo_text_appendf(&demo->report, message, message_size, "summary: %d parameters, %d mock stack instructions\n", demo->parameters, demo->commands)) {
        return 0;
    }
    return 1;
}

static int dks_run_assertions(const char *source, char *message, size_t message_size) {
    dks_demo demo = {0};
    datakeeper_error error;
    datakeeper_lexeme *tokens = NULL;
    size_t count = 0;
    if (!dks_parse(&demo, source, message, message_size)) {
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return 0;
    }
    if (demo.parameters != 4 || demo.commands != 8) {
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "unexpected summary: %d parameters, %d commands", demo.parameters, demo.commands);
    }
    demo_text_free(&demo.report);
    demo_arena_free(&demo.arena);
    if (datakeeper_tokenize("begin @ end", &tokens, &count, &error)) {
        datakeeper_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure");
    }
    if (!datakeeper_tokenize("begin end", &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (datakeeper_parse(tokens, count, &error)) {
        datakeeper_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure");
    }
    datakeeper_free_lexemes(tokens);
    return 1;
}

static const char *dks_read_option(int *argc, char **argv, const char *name, const char *fallback) {
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

static int dks_take_flag(int *argc, char **argv, const char *name) {
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
    dks_demo demo = {0};
    int assert_mode = dks_take_flag(&argc, argv, "--assert");
    const char *log_path = dks_read_option(&argc, argv, "--log", "dist/datakeeper-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "sample.dks";
    if (!demo_read_file(input_path, &source, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (assert_mode && !dks_run_assertions(source.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        return 1;
    }
    if (!dks_parse(&demo, source.data, message, sizeof(message)) ||
        !demo_write_text(log_path, demo.report.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return 1;
    }
    printf("%s", demo.report.data);
    demo_free_buffer(&source);
    demo_text_free(&demo.report);
    demo_arena_free(&demo.arena);
    return 0;
}
